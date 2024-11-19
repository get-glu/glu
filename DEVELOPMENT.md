# Development

## Requirements

- [Go](https://go.dev/) (version 1.23+)
- [Node.js](https://nodejs.org/) (version 20+) (optional, only if you want to build the UI)

## Glu Framework

The core framework is implemented in Go and is designed to be embedded in your own application.
Glu's documentation site goes into details for integrating it into your own codebase and learning the concepts.

If you want to contribute to Glu, then Go (version 1.23+) is currently the only requirement to get building.

## Glu UI

The Glu UI is a [React](https://react.dev/) application that allows you to view and interact with the state of your pipelines.

```
cd ui
npm install
npm start
```

This will start a local server which can be viewed in your browser at http://localhost:1234.

## Conventional Commits

Flipt uses [Conventional Commits](https://www.conventionalcommits.org/en/v1.0.0/) for commit messages. This allows us to automatically generate changelogs and releases. To help with this, we use [pre-commit](https://pre-commit.com/) to automatically lint commit messages. To install pre-commit, run:

`pip install pre-commit` or `brew install pre-commit` (if you're on a Mac)

Then run `pre-commit install` to install the git hook.
