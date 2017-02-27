# GH Blame

A simple tool to calculate average response times on issues and pull requests for a given Github repository.

Usage:

```
go run main.go <GitHub token> <repo owner> <repo>
```

Example:

```
go run main.go <token> bolt bolt
```

The example above will fetch the latest 100 closed issues and pull requests for `bolt/bolt` and return the following output:

```
For 100 Issues:
Average time until first comment: 6538 mins
Average time until close: 18827 mins

For 100 Pull Requests:
Average time until first comment: 314 mins
Average time until merge: 547 mins
Average time until close: 528 mins
```
