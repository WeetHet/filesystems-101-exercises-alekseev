#![no_std]

use core::ffi::CStr;

use rustix::fd::BorrowedFd;
use rustix::io::Errno;

/// Implement this function to copy the content of a file at `path`
/// to a file descriptor `out`. `path` has no symlinks inside it.
/// `img` is a file descriptor of an open ext2 image.
///
/// It suffices to support single- and double-indirect blocks.
///
/// If a copy was successful, return `Ok(())`. If an error occurred during
/// a read or a write, return `Err(errno)`.
///
/// Do take care to return `Errno::NOENT`, `Errno::NOTDIR` and other errors that
/// may happen during a path traversal.
pub fn dump_file(img: BorrowedFd<'_>, path: &CStr, out: BorrowedFd<'_>) -> Result<(), Errno> {
    let _ = (img, path, out);
    todo!()
}
