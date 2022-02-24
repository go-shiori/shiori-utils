# Sqlite migration to FTS5

Shiori utility to migrate the SQLite database from FTS4 to FTS5.

**This upgrade is mandatory for all users coming from 1.5.0 considering upgrading to 1.5.1+.**

The reason of this is that the SQLite dependency used in Shiori was changed in order to streamline the build system and make easier cross-compiling to other platforms. This can be seen in both the container registry and the releases page, providing automatically builds for systems we couldn't before.

Sadly, in this upgrade the full-text search of was updated directly as well, which is not a problem for new users but it comes with a critical (non-destructive) bug for old users: since the old table is created using fts4 and the new library does not support it, old users couldn't manage their bookmarks in any way:

```
Failed to get bookmarks: failed to fetch data: no such module: fts4
```

After trying a few approaches, the new SQLite library is a step forward regarding shiori maintenance and progress and instead of rolling back I decided to make a separate simple tool to migrate the table from fts4 to fts5, so users can upgrade their databases and continue using shiori's latest versions.

Why not making the migration directly in Shiori? That would require going back to the CGO version of Sqlite, compiling it to use fts5, migrate the table, and then switch back to the non-C variant. This way the version that make the migration would be mandatory (you could not skip it). Instead of the headaches all of those represent I'd rather have this separate and link to it from the releases page.

## Usage

:warning: Remember to make a backup of your `shiori.db` file in case something goes wrong.

```
$ shiori-sqlite-fts4-to-fts5 -path /path/to/database/shiori.db
```

Or running directly the code

```
go run --tags fts5 main.go -path /path/to/database/shiori.db
```

## Building

Remember to build with the `fts5` tag to gain fts4 and fts5 support (which is obviously required for this migration tool to work).

```
go build --tags fts5
```
