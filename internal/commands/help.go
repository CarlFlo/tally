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
`

func printHelp() {
	fmt.Print(helpText)
}
