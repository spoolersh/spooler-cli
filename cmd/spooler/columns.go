package main

import "github.com/spoolersh/spooler-cli/internal/format"

// queueColumns is what a text listing shows of a queue.
var queueColumns = must(format.ColumnsFor[queueOutput](
	"name",
	"state",
	"stats.visible",
	"stats.pending",
	"stats.failed",
	"createdAt",
))

// messageColumns is what a table listing shows of a message; the listing
// drops the ones its flags leave empty.
var messageColumns = must(format.ColumnsFor[messageOutput](
	"id",
	"state",
	"retries",
	"createdAt",
	"visibleAt",
	"leaseExpiresAt",
	"expiresAt",
	"handle",
))
