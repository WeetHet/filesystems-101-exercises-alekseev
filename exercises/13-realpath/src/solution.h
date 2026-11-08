#pragma once

/**
   Implement this function to expand all symlinks in @path and
   convert it to an absolute path that points to the same file
   or directory as @path does. It must call report_path() when
   all replacements are made. If an error occurs while walking
   the path, call report_error() and exit.

   When dealing with relative paths you may assume that the
   current working directory is "/".

   NOTE: paths to directories must end with '/' to make it clear
   that a path points to a directory.

   Examples:
   1. abspath("/usr/bin/cat")   -> report_path("/usr/bin/cat"),
   2. abspath("/usr/bin")       -> report_path("/usr/bin/"),
   3. abspath("does_not_exist") -> report_error("/", "does_not_exist", ENOENT).
 */
void abspath(const char *path);

/**
   abspath() must call this function once it resolves the path
   to an absolute one with all symlinks removed.

   @path is the result of the conversion to an absolute path.
 */
void report_path(const char *path);

/**
   abspath() must call this function if it detects an error
   while walking the path.

   @parent is the absolute path to a directory whose child
   could not be walked into.
   @child is the name of a child that could not be looked up
   (for example, one that does not exist).
   @errno_code is the (positive) errno code that describes
   why the path walk failed.
 */
void report_error(const char *parent, const char *child, int errno_code);
