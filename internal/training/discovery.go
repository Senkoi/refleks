package training

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"path"
	"regexp"
	"sort"
	"strings"
	"time"

	appsettings "aimmeow/internal/settings"
)

const maxSourceBytes = 2 << 20

var sharecodePattern = regexp.MustCompile(`\bKovaaKs[A-Za-z0-9]{8,100}\b`)
var jsonLinkPattern = regexp.MustCompile(`https://[^\s<>"()]+\.(?:json|plo)(?:\?[^\s<>"()]*)?`)

// Only public HTTPS sources are accepted; redirects receive the same validation.
func publicURL(ctx context.Context, raw string) error {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "https" || u.Hostname() == "" || u.User != nil || (u.Port() != "" && u.Port() != "443") {
		return fmt.Errorf("来源必须是公开 HTTPS 地址")
	}
	// These exact, built-in HTTPS endpoints can be resolved by an enterprise
	// proxy when local DNS is unavailable. Arbitrary user/web URLs still require
	// public address validation below; no wildcard or private host exception.
	if u.Hostname() == "api.github.com" || u.Hostname() == "raw.githubusercontent.com" || u.Hostname() == "api.search.brave.com" {
		return nil
	}
	ips, err := net.DefaultResolver.LookupIPAddr(ctx, u.Hostname())
	if err != nil || len(ips) == 0 {
		return fmt.Errorf("无法解析来源主机")
	}
	for _, ip := range ips {
		if !ip.IP.IsGlobalUnicast() || ip.IP.IsPrivate() || ip.IP.IsLoopback() || ip.IP.IsLinkLocalUnicast() {
			return fmt.Errorf("不读取本机或内网来源")
		}
	}
	return nil
}

func fetch(ctx context.Context, raw string, headers map[string]string) ([]byte, error) {
	if err := publicURL(ctx, raw); err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, raw, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "AimMeow-Training/0.11")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	client := &http.Client{Timeout: 12 * time.Second, CheckRedirect: func(r *http.Request, via []*http.Request) error {
		if len(via) >= 4 {
			return fmt.Errorf("来源重定向过多")
		}
		// Never forward a search credential to a redirect target.
		r.Header.Del("X-Subscription-Token")
		return publicURL(r.Context(), r.URL.String())
	}}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("来源返回 HTTP %d", resp.StatusCode)
	}
	b, err := io.ReadAll(io.LimitReader(resp.Body, maxSourceBytes+1))
	if err != nil {
		return nil, err
	}
	if len(b) > maxSourceBytes {
		return nil, fmt.Errorf("来源超过 2 MB 限制")
	}
	return b, nil
}

func rawGitHubURL(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return raw
	}
	if u.Hostname() == "github.com" {
		parts := strings.Split(strings.Trim(u.Path, "/"), "/")
		if len(parts) >= 5 && parts[2] == "blob" {
			return "https://raw.githubusercontent.com/" + parts[0] + "/" + parts[1] + "/" + strings.Join(parts[3:], "/")
		}
	}
	return raw
}

func readSource(ctx context.Context, raw, title string) ([]Scenario, Candidate, error) {
	data, err := fetch(ctx, rawGitHubURL(raw), nil)
	c := Candidate{Title: title, URL: raw, Sharecodes: []string{}}
	if err != nil {
		return nil, c, err
	}
	scenarios, parseErr := ParsePlaylist(data, Source{URL: raw, Title: title})
	if parseErr == nil {
		return scenarios, c, nil
	}
	seen := map[string]bool{}
	for _, code := range sharecodePattern.FindAllString(string(data), 100) {
		if !seen[code] {
			c.Sharecodes = append(c.Sharecodes, code)
			seen[code] = true
		}
	}
	c.Description = "公开页面线索；分享码尚未解析成关卡，不参与自动编排。"
	return nil, c, nil
}

// discover uses actual playlist files as machine-readable candidates and an optional
// user-owned Brave Search key for broader web discovery. It never guesses names from prose.
func discover(ctx context.Context, focus string) ([]Scenario, Discovery) {
	d := Discovery{Updated: time.Now().UTC().Format(time.RFC3339), Candidates: []Candidate{}, Warnings: []string{}}
	all := []Scenario{}
	data, err := fetch(ctx, "https://api.github.com/repos/riddbtw/kovaaks-playlists/git/trees/main?recursive=1", nil)
	if err == nil {
		var tree struct {
			Tree []struct {
				Path string `json:"path"`
			} `json:"tree"`
		}
		if e := json.Unmarshal(data, &tree); e != nil {
			d.Warnings = append(d.Warnings, "资源索引格式异常")
		} else {
			paths := []string{}
			for _, r := range tree.Tree {
				if strings.HasPrefix(r.Path, "routines-recommended/") && (strings.HasSuffix(r.Path, ".json") || strings.HasSuffix(r.Path, ".plo")) {
					paths = append(paths, r.Path)
				}
			}
			rand.Shuffle(len(paths), func(i, j int) { paths[i], paths[j] = paths[j], paths[i] })
			if len(paths) > 6 {
				paths = paths[:6]
			}
			for _, p := range paths {
				if ctx.Err() != nil {
					break
				}
				u := "https://raw.githubusercontent.com/riddbtw/kovaaks-playlists/main/" + strings.ReplaceAll(url.PathEscape(p), "%2F", "/")
				s, _, e := readSource(ctx, u, path.Base(p))
				if e != nil {
					d.Warnings = append(d.Warnings, path.Base(p)+": "+e.Error())
				} else {
					all = mergeCatalog(all, s)
				}
			}
		}
	} else {
		d.Warnings = append(d.Warnings, "公开列表索引："+err.Error())
	}
	// Discover recently maintained public repositories beyond the built-in
	// starting points. A repository is a candidate source, not a quality vote.
	repoQuery := "kovaaks playlist in:name,description"
	data, err = fetch(ctx, "https://api.github.com/search/repositories?per_page=5&sort=updated&q="+url.QueryEscape(repoQuery), nil)
	if err != nil {
		d.Warnings = append(d.Warnings, "GitHub 公开仓库搜索："+err.Error())
	} else {
		var result struct {
			Items []struct {
				FullName      string `json:"full_name"`
				DefaultBranch string `json:"default_branch"`
				URL           string `json:"html_url"`
				Description   string `json:"description"`
			} `json:"items"`
		}
		if e := json.Unmarshal(data, &result); e != nil {
			d.Warnings = append(d.Warnings, "GitHub 仓库搜索格式异常")
		} else {
			for _, repo := range result.Items {
				if ctx.Err() != nil {
					break
				}
				if repo.FullName == "" || repo.DefaultBranch == "" || strings.Count(repo.FullName, "/") != 1 {
					continue
				}
				d.Candidates = append(d.Candidates, Candidate{Title: repo.FullName, URL: repo.URL, Description: "公开仓库线索；关卡质量、版本和适用性待核验。 " + repo.Description})
				if repo.FullName == "riddbtw/kovaaks-playlists" {
					continue
				}
				parts := strings.Split(repo.FullName, "/")
				treeURL := "https://api.github.com/repos/" + url.PathEscape(parts[0]) + "/" + url.PathEscape(parts[1]) + "/git/trees/" + url.PathEscape(repo.DefaultBranch) + "?recursive=1"
				body, e := fetch(ctx, treeURL, nil)
				if e != nil {
					continue
				}
				var tree struct {
					Truncated bool `json:"truncated"`
					Tree      []struct {
						Path string `json:"path"`
						Type string `json:"type"`
					} `json:"tree"`
				}
				if json.Unmarshal(body, &tree) != nil || tree.Truncated {
					continue
				}
				paths := []string{}
				for _, file := range tree.Tree {
					lower := strings.ToLower(file.Path)
					if file.Type == "blob" && len(file.Path) <= 250 &&
						(strings.HasSuffix(lower, ".json") || strings.HasSuffix(lower, ".plo")) &&
						(strings.Contains(lower, "playlist") || strings.Contains(lower, "routine")) {
						paths = append(paths, file.Path)
					}
				}
				sort.Strings(paths)
				if len(paths) > 2 {
					paths = paths[:2]
				}
				for _, filePath := range paths {
					if ctx.Err() != nil {
						break
					}
					link := "https://raw.githubusercontent.com/" + url.PathEscape(parts[0]) + "/" + url.PathEscape(parts[1]) + "/" + url.PathEscape(repo.DefaultBranch) + "/" + strings.ReplaceAll(url.PathEscape(filePath), "%2F", "/")
					items, _, e := readSource(ctx, link, repo.FullName+" / "+path.Base(filePath))
					if e == nil {
						all = mergeCatalog(all, items)
					}
				}
			}
		}
	}
	_, candidate, err := readSource(ctx, "https://raw.githubusercontent.com/4BangerKovaaks/kovaaks-playlist-compendium/main/README.md", "4BK 训练资源库")
	if err == nil {
		d.Candidates = append(d.Candidates, candidate)
	} else {
		d.Warnings = append(d.Warnings, "4BK："+err.Error())
	}
	key := appsettings.GetEnv("AIMMEOW_BRAVE_API_KEY")
	if key == "" {
		d.Warnings = append(d.Warnings, "这次先帮你找公开训练资源喵。配置搜索服务后，我还能找到更多作者页面。")
		return all, d
	}
	query := "Kovaaks " + focus + " aim training routine playlist new scenarios"
	data, err = fetch(ctx, "https://api.search.brave.com/res/v1/web/search?count=8&q="+url.QueryEscape(query), map[string]string{"X-Subscription-Token": key, "Accept": "application/json"})
	if err != nil {
		d.Warnings = append(d.Warnings, "全网搜索："+err.Error())
		return all, d
	}
	var result struct {
		Web struct {
			Results []struct {
				Title       string `json:"title"`
				URL         string `json:"url"`
				Description string `json:"description"`
			} `json:"results"`
		} `json:"web"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		d.Warnings = append(d.Warnings, "搜索结果格式异常")
		return all, d
	}
	for i, r := range result.Web.Results {
		if i >= 8 || ctx.Err() != nil {
			break
		}
		b, e := fetch(ctx, rawGitHubURL(r.URL), nil)
		c := Candidate{Title: r.Title, URL: r.URL, Description: r.Description, Sharecodes: []string{}}
		if e == nil {
			if ss, e := ParsePlaylist(b, Source{URL: r.URL, Title: r.Title}); e == nil {
				all = mergeCatalog(all, ss)
			}
			c.Sharecodes = sharecodePattern.FindAllString(string(b), 20)
			// Follow at most two explicit playlist links per page, never arbitrary links.
			for _, link := range jsonLinkPattern.FindAllString(string(b), 2) {
				if ss, _, e := readSource(ctx, link, r.Title); e == nil {
					all = mergeCatalog(all, ss)
				}
			}
		} else {
			d.Warnings = append(d.Warnings, r.Title+": 无法读取正文，保留搜索线索")
		}
		d.Candidates = append(d.Candidates, c)
	}
	return all, d
}
