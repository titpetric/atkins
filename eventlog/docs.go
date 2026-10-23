// Package eventlog records what one atkins run does: an event per step,
// per $() substitution and per interpolated command, the state tree the
// run ends with, and a summary of the totals.
//
// A Logger collects the events of one run. It collects with or without a
// file: the storage recorder reads the events and the summary counts
// them, and a collected event costs one slice append. The file path
// decides whether Write writes the YAML document of --log.
//
// Document assembles the finished Log. Write marshals it and the storage
// recorder stores it, so the file and the event database read the same
// document.
package eventlog
