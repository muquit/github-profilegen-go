## Table Of Contents
- [Introduction](#introduction)
- [Features](#features)
- [Synopsis](#synopsis)
- [Version](#version)
- [Installation](#installation)
  - [Install from github](#install-from-github)
  - [Download](#download)
  - [Building from source](#building-from-source)
- [Usage](#usage)
- [Configuration Files](#configuration-files)
  - [Exclusion List (exclude.txt)](#exclusion-list-excludetxt)
  - [Priority List (priority.txt)](#priority-list-prioritytxt)
  - [AI Credits (ai-credits.txt)](#ai-credits-ai-creditstxt)
  - [Contact Information (contact.txt)](#contact-information-contacttxt)
- [Example Output](#example-output)
- [License is MIT](#license-is-mit)
- [Authors](#authors)

# Introduction

`github-profilegen-go` is a simple multi-platform tool to generate clean, 
minimal GitHub profile README.md for your repositories.
It can be usefull if you want to list all your repositories instead of 
default pinned 6 repositories.

- Please visit https://github.com/muquit/ to see how it looks like.
- Please vist https://github.com/muquit/muquit/ and look at `Makefile` to
  see how the [README.md](https://github.com/muquit/muquit/) is generated.

# Features
- Creates a card-based layout for repositories
- Displays repository name, description, language, dates, and fork status
- Supports priority ordering of repositories
- Allows excluding specific repositories
- Includes optional contact information
- Badges for language, starts, forks, downloads etc.

# Synopsis
```
Usage of ./github-profilegen-go:
  -ai-credits string
    	Path to AI credits file
  -contact string
    	Path to contact info file
  -exclude string
    	Path to exclusion list file
  -output string
    	Path to output file (default "README.md")
  -priority string
    	Path to priority list file
  -user string
    	GitHub username (required)
  -version
    	Show version information and exit
```

# Version
The current version is 1.0.3

Please look at [ChangeLog](ChangeLog.md) for what has changed in the current version.

# Installation
## Install from github

Install [go](https://go.dev/) first

```bash
go install github.com/yourusername/github-profilegen-go@latest
```

## Download

Download pre-compiled binaries for various platforms from
[Releases](https://github.com/muquit/github-profilegen-go/releases) page

## Building from source
Install [go](https://go.dev/) first

```bash
git clone https://github.com/yourusername/github-profilegen-go.git
cd github-profilegen-go
go build
```

Look at `Makefile` for more info

# Usage

In github, create a repository with your username. It is a special repository. Add the
generated `README.md` in this repo.

- Please look at  github document for details:
[Managing your profile README](https://docs.github.com/en/account-and-profile/setting-up-and-managing-your-github-profile/customizing-your-profile/managing-your-profile-readme)

- Please vist https://github.com/muquit/muquit/ and look at `Makefile` to
  see how the [README.md](https://github.com/muquit/muquit/) is generated.

```bash
github-profilegen-go -user=yourusername
github-profilegen-go -user=yourusername \
    -exclude exclude.txt \
    -priority priority.txt \
    -contact contact.txt
```
If your project is generated with AI assistance and wants to give credit, use
the flag:

```
    -ai-credits ai-credits.txt
```

`README.md` will be created

# Configuration Files

## Exclusion List (exclude.txt)
Contains names of repositories to exclude from the README, one per line:
```
test-repo
personal-notes
old-project
```

## Priority List (priority.txt)
Contains repository names in the order they should appear at the top of the README:
```
important-project
cool-library
useful-tool
```

## AI Credits (ai-credits.txt)
If a project is AI assisted and if you would like to give credit, specify a
file with the options `-ai-credits ai-credits.txt`.

An Example file is shown below:
```
# Format: repository_name|image_path|alt_text|title_text|width|height
go-xbuild-go|claude_assisted.svg|Claude AI Assisted|Claude AI Assisted|96|31
cubic-ubuntu-server-fix|claude_assisted.svg|Claude AI Assisted|Claude AI Assisted|96|31
```
Look at `ai-credits.txt` I use for my github profile README.md. Look at
https://github.com/muquit to see how it looks like.

## Contact Information (contact.txt)
Contains contact details to be displayed in a Contact section:
```
📧 Email: your.email@example.com
🌐 Website: https://yourwebsite.com
💼 LinkedIn: https://linkedin.com/in/yourprofile
```
You can add anything in this file will will show up at the end of README.md


# Example Output
The generated README will display repository cards with:
- Repository name and link
- Description
- Programming language
- Creation date
- Last update date
- Publication date
- Fork status

Look at [README.md](https://github.com/muquit/muquit/) for an example.

# License is MIT

See LICENSE.txt file for details.

# Authors
- Developed with Claude AI 3.7 Sonnet, working under my guidance and instructions. 
- Updated with Google Gemini AI 2.5 Pro (May-24-2025)

---
<sub>TOC is created by https://github.com/muquit/markdown-toc-go on May-24-2025</sub>
