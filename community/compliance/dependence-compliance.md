# Dependencies Compliance

This document aims to outline the compliance for dependencies in Kurator.

## Background

Kurator uses Go modules to track dependencies.

Go modules allow recording desired versions of dependencies, and allow the main
module in a build to pin dependencies to specific versions.

## Justifications for an update

Before you update a dependency, take a moment to consider why it should be updated. Valid reasons include:

- We need new functionality that is in a later version.
- New or improved APIs in the dependency significantly improve Kurator code.
- Bugs were fixed that impact Kurator.
- Security issues were fixed even if they don't impact Kurator yet.
- Performance, scale, or efficiency was meaningfully improved.
- We need dependency A and there is a transitive dependency B.
- Kurator has an older level of a dependency that is precluding being able

to work with other projects in the ecosystem.

## Dependency versions

As a project we prefer that all entries in `go.mod` should be tagged in their respective repositories.
There may be exceptions that will be up to the dependency approvers to approve.
If there are issues with go mod tooling itself then there has to be an explicit comment (trailing `// comment`) with details on exact tag/release that this SHA corresponds to.
Also please ensure tracking issues are open to ensure these SHA(s) are cleaned up over time and switched over to tags.

## Commit messages

Terse messages like "Update foo.org/bar to 0.42" are problematic for maintainability.
Please include in your commit message the detailed reason why the dependencies were modified.

Too commonly dependency changes have a ripple effect where something else breaks unexpectedly.
The first instinct during issue triage is to revert a change.
If the change was made to fix some other issue and that issue was not documented, then a revert simply continues the ripple by fixing one issue and reintroducing another which then needs refixed.
This can needlessly span multiple days as CI results bubble in and subsequent patches fix and refix and rerefix issues.
This may be avoided if the original modifications recorded artifacts of the change rationale.

## Reviewing and approving dependency changes

Particular attention to detail should be exercised when reviewing and approving
PRs that add/remove/update dependencies. Importing a new dependency should bring
a certain degree of value as there is a maintenance overhead for maintaining
dependencies into the future.

When importing a new dependency, be sure to keep an eye out for the following:

- Is the dependency maintained?
- Does the dependency bring value to the project? Could this be done without
  adding a new dependency?
- Is the target dependency the original source, or a fork?
- Is there already a dependency in the project that does something similar?
- Does the dependency have a license that is compatible with the Kurator
  project?

Additionally:

- Look at the `go.mod` changes in `kurator.dev/kurator`.
  Check that the only changes are what the PR claims them to be.
- Look at the `go.mod` changes in the staging components.
  Avoid adding new `replace` directives in staging component `go.mod` files.
  New `replace` directives are problematic for consumers of those libraries,
  since it means we are pinned to older versions than would be selected by go
  when our module is used as a library.
- Check if there is a tagged release we can vendor instead of a random hash
- Scan the imported code for things like init() functions
- Look at the Kurator code changes and make sure they are appropriate
  (e.g. renaming imports or similar). You do not need to do feature code review.
- If this is all good, approve, but don't LGTM, unless you also do code review
  or unless it is trivial (e.g. moving from k/k/pkg/utils -> k/utils).

Licenses for dependencies are specified by the Kurator [allowed-licenses-list](/common/config/license-lint.yaml).
All new dependency licenses should be reviewed by @[kurator/dep-approvers] to ensure that they
are compatible with the Kurator project license. It is also important to note
and flag if a license has changed when updating a dependency, so that these can
also be reviewed.

In case of questions or concerns regarding the allowlist policy, please create
an issue or send a message to the member of [kurator/dep-approvers].

## Licences restrictions

In the Kurator project, there are compliance requirements for the licenses of dependencies used. We prohibit the use of dependencies with infectious licenses. You can check [allowed-licenses-list](/common/config/license-lint.yaml) to learn about Kurator project's specifications on license compliance.

It specifies that licenses listed in the "restrictions" section cannot be used in the kurator project. Licenses in the "reciprocal_licenses" section can be used but modifications are not permitted. Prohibition of licences in the "restricted_licenses" section.

If you need to use a license that is not included in either section, please open a [issues](https://github.com/kurator-dev/kurator/issues) for discussion or send a message to the member of [kurator/dep-approvers].

If you have any questions regarding licenses compliance, you can contact the [kurator/dep-approvers] as well.

[kurator/dep-approvers]: dep-approvers.md
