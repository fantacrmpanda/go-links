
**go-links** is a stand-along server that lets you set a keyword to get to an
URL.  This is not a URL shortener which takes a URL that generates a shorter
link, but lets you specify a short and easy-to-remember word to use so that if
you type something like `go/hr` then it might direct you to your company's HR
site without you having to remember the exact URL.

This is often used by companies and teams to make it easier to share links and
is often referred to as "go links" because Google (which is generally considered
the place where it originated) used `go` as the hostname (as in `go/hr`) since
it is short and fast to type, but another hostname can be used.

## Background

For my company, I really wanted to keep on using go links.  There are some
companies that offer go links as a Software-as-a-Service (SaaS) and there are
open source solutions such as libraries so you can build your own version or
packages that you can host yourself.

I didn't want to use a SaaS because this is something everyone in the
team/company will use and that cost can add up fast.  I looked at open source
solutions but they didn't quite have what I needed and/or required more setup
then I wanted to invest in for my own simple needs.

## Features

My goals was to have something that provided the basic features of go links
along with very simple deployment and maintenance.


The basics features of go-links are:

* Save/Update/Delete a keyword and its corresponding URL.
* When user access the server with a known keyword, they get redirected to the
  URL.
* When user uses an unknown keyword, they get directed to a form to create a new
  go link.

Other features:

* Simple deployment - go-links is a single binary.
* Uses oAuth2 for user authentication (initially with Google accounts)
* Support sessions.
* Simple code so that people can easily modify it for their needs.
   * go-links is not meant to be library for building go link services.
* Simple to build and compile - go-links is built using the Go programming
  langauge (no relations) and uses all native Go code.

NOTE: go-links is not meant for large enterprises with 100k users.  I've not
optimized for that scale.  This is meant for my needs but thought it might be
helpful for others so I wanted to make the code available.

## Installation

1. Copy the go-link binary to wherever you want to run it from.

NOTE: If there is no sqlite file, the program will generate one and enable WAL
on it.  It will do basically the same as "sqlite3 data.db PRAGMA jounal_mode=WAL"

NOTE: This initial version only supports Google Accounts so you'll need to configure
the your Google oAuth client from http://console.developers.google.com and get
your client ID and secret as well as register your callback URLs to point to
your host.

## Building

Requirements:
  * go v 1.22+

In the main directory:

`go build .`

## Usage

```
export GOOGLE_CLIENT_ID=<your client ID from Google>
export GOOGLE_CLIENT_SECRET=<your client secret from Google>
export GOOGLE_CALLBACK_URL=<oAuth Callback URL that you set with Google>
<path-to-go-links>/go-links
```

You can chose which HTTP and HTTPS ports to use by passing to `go-links` the
port through command line parameters or environment variables.  `go-links`-help`
will show the different parameters.

## Browser Extension

I have a [simple browser
extension](https://github.com/lazyhacker/go-links-chrome-extension) to rewrite go/ to the go-link server if the
machine's host settings can't be modified.

## TODO

- memory cache
- link transfer
- refactor hard-coded strings

