# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.1.1] - 2024-01-24

### Fixed

- Bug when restoring a session from disk that caused SEGFAULT when updating session token
- Bug when creating a new session pulled the wrong value for remember-token
- Incorrect types when creating orders

## [0.1.0] - 2024-01-22

### Added

- Session management (create and delete session)
- List accounts
- Get account balance
- List account transactions
- List account positions
- List account orders
- Create new order (simple orders only)
- Delete existing order

[unreleased]: https://github.com/penny-vault/go-tasty/compare/v0.1.0...HEAD
[0.1.1]: [0.0.2]: https://github.com/penny-vault/go-tasty/compare/v0.1.0...v0.1.1
[0.1.0]: https://github.com/penny-vault/go-tasty/releases/tag/v0.1.0