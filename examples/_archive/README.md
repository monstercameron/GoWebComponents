# Archived Examples

This directory contains legacy examples that used the old `fiber` package HTML generation API which has been deprecated.

These files are kept for reference but are not currently maintained or compatible with the current GoWebComponents architecture.

## Files

- `concurrent_dashboard.go` - Real-time concurrent data processing dashboard (requires fiber HTML API)
- `network_monitoring_dashboard.go` - Network monitoring visualization dashboard (requires fiber HTML API)

## Migration Notes

To use similar functionality with the current architecture, use the `dom` package for HTML generation instead of the fiber package's dot-import HTML helpers.
