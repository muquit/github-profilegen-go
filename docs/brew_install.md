## Installing using Homebrew on Mac/Linux

You will need to install [Homebrew](https://brew.sh/) first.

### Install

First install the custom tap, then trust it. Homebrew 6.0+ refuses to load
formulae from third-party taps until they are explicitly trusted.

```
brew tap muquit/github-profilegen-go https://github.com/muquit/github-profilegen-go.git
brew trust muquit/github-profilegen-go
brew install github-profilegen-go
```

Or tap, trust and install in one go:
```
brew tap muquit/github-profilegen-go https://github.com/muquit/github-profilegen-go.git
brew trust muquit/github-profilegen-go
brew install muquit/github-profilegen-go/github-profilegen-go
```

### Upgrade
```
brew upgrade github-profilegen-go
```

### Uninstall
```
brew uninstall github-profilegen-go
```

### Remove the tap
```
brew untap muquit/github-profilegen-go
```
