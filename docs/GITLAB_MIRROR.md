# GitLab mirror and Harbor publish

The operator image is built and pushed to Harbor from GitLab CI on the mirror at
`gitlab.com/dk-raas/dkai/game-servers/windrose-operator`.

GitHub Actions on `main` runs verification, then mirrors the repository to GitLab.
GitLab CI runs tests, builds the Docker image, and publishes to
`harbor.dataknife.net/library/windrose-operator`.

## Credentials (already configured at org/group level)

No per-repo secrets or CI variables are required for normal operation.

### GitHub — org secret `GITLAB_TOKEN`

The DataKnifeAI organization provides `GITLAB_TOKEN` to repositories that mirror
to GitLab. Verify access for this repo:

```bash
gh api repos/DataKnifeAI/windrose-operator/actions/organization-secrets
```

Expected: `GITLAB_TOKEN` in the `secrets` list.

The **Mirror to GitLab** workflow reads `${{ secrets.GITLAB_TOKEN }}` on pushes
to `main`. Sibling repos (for example `windrose-server-k8s`) use the same
org secret successfully.

### GitLab — group variables on `dk-raas/dkai`

Harbor and registry credentials are inherited from the parent group. Verify:

```bash
glab variable list --group dk-raas/dkai
```

Expected variables include `HARBOR_REGISTRY`, `HARBOR_PROJECT`, `HARBOR_USERNAME`,
`HARBOR_PASSWORD`, and `DOCKER_AUTH_CONFIG`. The `game-servers` subgroup and
`windrose-operator` project do not need their own copies.

Non-secret values (safe to print):

```bash
glab variable get HARBOR_REGISTRY --group dk-raas/dkai
glab variable get HARBOR_PROJECT --group dk-raas/dkai
```

## Authenticate GitLab CLI (`glab`)

For manual pushes or project management:

```bash
glab auth login --hostname gitlab.com
glab auth status
```

## Mirror project

https://gitlab.com/dk-raas/dkai/game-servers/windrose-operator

Created once with:

```bash
glab repo create windrose-operator \
  --group dk-raas/dkai/game-servers \
  --private \
  --description "Mirror of DataKnifeAI/windrose-operator — CI builds operator image to Harbor"
```

## Image tags

Pushes produce `:latest`, `:<commit-sha>`, and `:<git-tag>` when applicable.

## Manual mirror push

If GitHub Actions has not run yet:

```bash
git remote add gitlab https://gitlab.com/dk-raas/dkai/game-servers/windrose-operator.git
git push gitlab main
```

Note: `main` on GitLab may be branch-protected; prefer merging to GitHub `main`
and letting the mirror workflow sync.
