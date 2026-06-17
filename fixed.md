# Fixed: shields.io badge errors in generated README

## Problem
Generated profile README showed "Unable to select next GitHub token from pool"
instead of Stars/Forks/Downloads badge images.

## Root cause
shields.io's *dynamic* badge endpoints (`img.shields.io/github/stars/...`,
`/forks/...`, `/downloads/.../total`) query the GitHub API using shields.io's
own internal pool of GitHub tokens. When that pool is rate-limited, shields.io
returns an error SVG as the image content (still a 200 response), which then
gets cached by GitHub's camo image proxy and shown to every viewer.

## Fix (main.go)
- Stars and Forks badges switched to static `img.shields.io/badge/...` URLs
  built from counts already fetched via the GitHub API (`Repository.Stargazers`,
  `Repository.ForksCount`). No shields.io-side GitHub call needed anymore.
- Downloads badge: added `fetchLatestRelease()` which paginates
  `/repos/{owner}/{repo}/releases` and sums `download_count` across all
  release assets (matching what shields.io's `/total` endpoint reported),
  storing it in new `Repository.TotalDownloads`. Badge switched to static
  `img.shields.io/badge/Downloads-{count}-green` using that value.
- Added `errRateLimited` sentinel: if GitHub's API rate limit is hit while
  fetching release data, the tool now aborts the whole run (exit 1) instead
  of silently writing a README with missing/zero download counts for the
  remaining repos. Re-run with `-token` / `GITHUB_TOKEN` set.

## Trade-off
Counts are now a snapshot from generation time, not live. Re-run the
generator periodically (e.g. weekly) to refresh. User confirmed this is fine.

## Separate, unrelated issue encountered while testing
After regenerating and pushing, some badges briefly appeared broken (showing
alt text as a link) on github.com. Diagnosed as GitHub's camo image proxy
(`camo.githubusercontent.com`) timing out (504) on first-fetch of a freshly
pushed image before anything is cached — confirmed by re-fetching the same
camo URLs and seeing inconsistent 200/504 results across requests, unrelated
to which repo or badge. Resolved on its own once camo cached successful
fetches (5-day cache, confirmed working after reload in Safari). Not a bug
in the generator; no code change made for this part.
