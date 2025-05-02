# MiniTwit
A project part of the DevOps course at the IT University of Copenhagen.

> [!NOTE]
> The report is in a different repository but is linked as a Git submodule. To clone the project along with the report, add the --recurse-submodules flag.


### Committing code
When committing code, we make use of the following following branching strategy:

1. Create a new branch from the `main` branch.
2. Commit your code to the new branch.
3. Merge into the `staging` branch to verify that the code works as expected.
4. Open a PR from staging to main and request reviews from our group members (at least two reviews are required).
5. After the PR is approved, merge the code into the `main` branch.

> [!NOTE]  
> Only working code should be merged into the `main` branch. If the code is not working as expected, it should be fixed before merging.

### Branches prefixes

To keep the branches organized, we will use the following prefixes:

| Branch purpose | Prefix |
|---|---|
| Main branch | `main` |
| Staging branch | `staging` |
| Refactor of feature | `refactor/` |
| Creating new feature | `feature/`|
| Feature enhancment | `enhancement/`|

