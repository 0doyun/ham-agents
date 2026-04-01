import Foundation

public enum ItermAppleScripts {
    public static func focusSession(_ sessionID: String) -> String {
        let escapedSessionID = appleScriptEscaped(sessionID)
        return """
        tell application "iTerm"
            activate
            repeat with aWindow in windows
                repeat with aTab in tabs of aWindow
                    repeat with aSession in sessions of aTab
                        if (id of aSession as string) is "\(escapedSessionID)" then
                            select aTab
                            select aSession
                            set index of aWindow to 1
                            return
                        end if
                    end repeat
                end repeat
            end repeat
            error "session not found" number 1
        end tell
        """
    }

    public static func writeToCurrentSession(_ message: String) -> String {
        let escapedMessage = appleScriptEscaped(message)
        return """
        tell application "iTerm"
            activate
            tell current window
                tell current session
                    write text "\(escapedMessage)"
                end tell
            end tell
        end tell
        """
    }

    public static func writeToSession(_ sessionID: String, message: String) -> String {
        guard !sessionID.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty else {
            return writeToCurrentSession(message)
        }

        let escapedSessionID = appleScriptEscaped(sessionID)
        let escapedMessage = appleScriptEscaped(message)
        return """
        tell application "iTerm"
            activate
            repeat with aWindow in windows
                repeat with aTab in tabs of aWindow
                    repeat with aSession in sessions of aTab
                        if (id of aSession as string) is "\(escapedSessionID)" then
                            tell aSession to write text "\(escapedMessage)"
                            return
                        end if
                    end repeat
                end repeat
            end repeat
            error "session not found" number 1
        end tell
        """
    }

    private static func appleScriptEscaped(_ text: String) -> String {
        text
            .replacingOccurrences(of: "\\", with: "\\\\")
            .replacingOccurrences(of: "\"", with: "\\\"")
    }
}
