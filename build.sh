#!/bin/sh

go build -ldflags "-X 'lazyhacker.dev/go-links/internal/buildinfo.buildinfo=`date`'"
