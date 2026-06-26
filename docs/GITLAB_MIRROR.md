# GitLab mirror and Harbor publish

The operator image is built and pushed to Harbor from GitLab CI on the mirror at
`gitlab.com/dk-raas/dkai/game-servers/windrose-operator`.

GitHub Actions on `main` runs verification, then mirrors the repository to GitLab.
GitLab CI runs `make ci`, builds the Docker image, and publishes to
`harbor.dataknife.net/library/windrose-operator`.

## Authenticate GitLab CLI (`glab`)

1. Create a [GitLab personal access token](https://docs.gitlab.com/user/profile/personal_access_tokens/)
   or [group/project access token](https://docs.gitlab.com/user/project/settings/project_access_tokens/)
   with at least **write_repository**.

2. Log in:

   ```bash
   glab auth login --hostname gitlab.com
   ```

   Or for non-interactive use:

   ```bash
   export GITLAB_TOKEN="glpat-..."   # do not commit this value
   ```

3. Verify:

   ```bash
   glab auth status
   ```

## Create the mirror project (once)

Already created for this repo:

```bash
glab repo create "dk-raas/dkai/game-servers/windrose-operator" \
  --private \
  --description "Mirror of DataKnifeAI/windrose-operator — CI builds operator image to Harbor"
```

## GitHub repository secret

Add a `GITLAB_TOKEN` secret on the GitHub repository with push access to the
mirror project. The **Mirror to GitLab** workflow uses it only to push `main`.

## GitLab CI variables (Harbor)

On the GitLab project (or parent group), set **masked** variables:

| Variable | Example |
|----------|---------|
| `HARBOR_REGISTRY` | `harbor.dataknife.net` |
| `HARBOR_PROJECT` | `library` |
| `HARBOR_USER` | Harbor robot or user |
| `HARBOR_PASSWORD` | Harbor token (masked, protected) |

Pushes produce tags `:latest`, `:<commit-sha>`, and `:<git-tag>` when applicable.

## Manual mirror push

If GitHub Actions is not yet configured:

```bash
git remote add gitlab https://gitlab.com/dk-raas/dkai/game-servers/windrose-operator.git
git push gitlab main --force
```
