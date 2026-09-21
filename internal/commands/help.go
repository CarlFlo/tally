package commands

import "fmt"

const helpText = `Tally command-line tools

Usage:
  tally <command> [arguments]

Commands:
  serve                               Start the Tally server (default)
  backup                              Create a manual backup
  restore [backup.zip]                Restore a backup; omit filename to list backups
  verify-backup [backup.zip]          Verify a backup; omit filename to list backups
  reset-password [profile-id-or-name] Reset a password; omit profile to list profiles
  delete-backup <filename.zip>        Delete a backup
  help                                Show this help

Backup files:
  Backup filenames are resolved from /config/backups by default.
  restore and verify-backup also accept an absolute archive path.

Examples:
  tally backup
  tally verify-backup tally-manual-20260921-180000-a1b2c3d4.zip
  tally restore tally-manual-20260921-180000-a1b2c3d4.zip
  tally reset-password
`

func printHelp() {
	fmt.Print(helpText)
}
