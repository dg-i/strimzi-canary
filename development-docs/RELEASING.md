# Releasing Strimzi Canary

This document describes how to release a new version of the Strimzi Canary.

## Regular releases

### Create release branch

Before releasing new major or minor version of Canary, the release branch has to be created.
The release branch should be named as `release-<Major>.<Minor>.x`.
For example for release 0.1.0, it should be named `release-0.1.x`.
The release branch is normally created from the `main` branch.
This is normally done locally and the branch is just pushed into the GitHub repository.

When releasing a new patch version, the release branch should already exist.
You just need to cherry-pick bug fixes or add them through PRs.

### Prepare the release

For any new release - major, minor or patch - we need to prepare the release.
The release preparation includes updating the installation files for the new version or changing the version of the project.
This is implemented in the `Makefiles`, you just need to run the command `RELEASE_VERSION=<NewRealeaseVersion> make release`.
For example, for release 0.1.0, you would run `RELEASE_VERSION=0.1.0 make release`.

Review and commit the changes done by the `make` command and push them into the repository.
The `Build` workflow should automatically start for any new commit pushed into the release branch.

### Running the release workflow

Wait until the `Build` workflow is (successfully) finished for the last commit in the release branch.
Then run the `Release` workflow manually from the GitHub Actions UI (or with `gh workflow run release.yaml --ref release-<Major>.<Minor>.x -f releaseVersion=<version>`), selecting the release branch.
When starting the new run, it will ask for the following parameters:

* Release version (for example `0.1.0`)
* Release suffix (optional, for example `0` - it is used to create an additional suffixed image such as `ghcr.io/dg-i/strimzi-canary:0.1.0-0` to identify different builds)

The release workflow builds the binaries for `linux/amd64` and `linux/arm64` from the release branch and pushes the multi-arch images to `ghcr.io/dg-i/strimzi-canary`.
It also creates a draft GitHub release, targeting the release branch commit, with the ZIP and TAR.GZ archives of the installation files attached.

### Smoke tests

After the release workflow is finished, it is always good idea to do some smoke tests of the images to double check they were pushed correctly.

### Creating the release

After the smoke tests, review and publish the draft GitHub release created by the release workflow.
Publishing it creates the release tag (e.g. `0.1.0`) on the release branch commit.

### Announcements

Announce the release on following channels:

* Mailing lists
* Slack
* Twitter (if the release is significant enough)

### Release candidates

Release candidates are built with the same release workflow as the final releases.
When starting the workflow, use the RC name as the release version.
For example `0.1.0-rc1` or `0.1.0-rc2`.
For release candidates, you should leave the release suffix empty since the suffixed build is not needed.
The draft GitHub release is marked as pre-release automatically.

When doing the release candidates, the release branch should be already prepared for the final release.
E.g. when building `0.1.0-rc1`, the release branch should have already the `0.1.0` versions set.
The release candidate version (e.g. `0.1.0-rc1`) should be used for the GitHub tag and release.
