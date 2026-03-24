# HamMenuBarApp

This directory is reserved for the macOS menu bar application.

It is intentionally kept outside the initial SwiftPM build graph so the repository can stabilize the Swift UI shell while the Go CLI/runtime/daemon foundation matures.

Expected responsibilities:

- menu bar status surface
- popover agent overview
- detail panel and future pixel-office UI
- app lifecycle integration with daemon snapshots and commands
