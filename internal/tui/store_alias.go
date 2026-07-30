package tui

import "github.com/ankurCES/Flup/internal/history"

// HistoryStore is the TUI-side alias so other tui files don't need to
// import internal/history directly. Keeps the import graph shallow.
type HistoryStore = history.Store
type HistoryEntry = history.Entry
