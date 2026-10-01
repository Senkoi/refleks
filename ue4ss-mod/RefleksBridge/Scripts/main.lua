-- Experimental KovaaK's runtime observer for UE4SS. It only reads challenge
-- state and appends lifecycle events. It does not change gameplay or scores.
-- Install under Mods/RefleksBridge/Scripts and enable RefleksBridge in mods.txt.
local home = os.getenv("USERPROFILE")
if not home then
    print("[RefleksBridge] USERPROFILE unavailable\n")
    return
end
local event_path = home .. "\\.refleks\\kovaaks-events.jsonl"
local previous_active = false
local previous_name = nil
local previous_remaining = nil
local last_end = nil

local function quoted(value)
    return '"' .. tostring(value):gsub('[%z\1-\31\\"]', function(c)
        local escapes = { ['\\'] = '\\\\', ['"'] = '\\"', ['\n'] = '\\n', ['\r'] = '\\r', ['\t'] = '\\t' }
        return escapes[c] or string.format('\\u%04x', c:byte())
    end) .. '"'
end

local function emit(kind, name)
    local file = io.open(event_path, "a")
    if not file then return end -- RefleK's creates its config directory on startup.
    file:write('{"type":', quoted(kind), ',"scenario":', quoted(name), ',"at":', tostring(os.time() * 1000), '}\n')
    file:close()
end

local function observe()
    local manager = FindFirstOf("ScenarioManager")
    if not manager then return end
    local ok, active = pcall(function() return manager:IsInChallenge() end)
    if not ok or type(active) ~= "boolean" then return end
    local name = nil
    for _, field in ipairs({ "CurrentScenario", "SelectedScenario", "ActiveScenario" }) do
        local found, value = pcall(function() return manager[field] end)
        if found and value then
            local named, result = pcall(function() return value:GetName() end)
            if named and type(result) == "string" and #result > 0 then name = result; break end
        end
    end
    if not name then return end -- Never attribute a run to a guessed scenario.
    local timed, remaining = pcall(function() return manager:GetChallengeTimeRemaining() end)
    if not timed or type(remaining) ~= "number" then remaining = nil end

    if active and (not previous_active or name ~= previous_name) then
        local restarted = last_end == "challenge_canceled" and name == previous_name
        emit(restarted and "challenge_restart" or "challenge_start", name)
        last_end = nil
    elseif active and previous_active and remaining and previous_remaining and remaining > previous_remaining + 3 then
        -- A countdown jumping back to its initial value is a restart signal.
        emit("challenge_restart", name)
    elseif not active and previous_active then
        last_end = remaining and remaining <= 1.5 and "challenge_complete" or "challenge_canceled"
        emit(last_end, previous_name or name)
    end
    previous_active, previous_name, previous_remaining = active, name, remaining
end

-- Poll Unreal objects on the game thread. Failure to resolve a field leaves
-- the observer idle; the desktop app still uses completed run files.
LoopAsync(1000, function()
    ExecuteInGameThread(function()
        local ok, err = pcall(observe)
        if not ok then print("[RefleksBridge] observation failed: " .. tostring(err) .. "\n") end
    end)
    return false
end)
