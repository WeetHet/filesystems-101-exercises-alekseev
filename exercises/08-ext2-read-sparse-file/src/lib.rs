#![no_std]

use core::ffi::c_int;

use rustix::fd::BorrowedFd;
use rustix::io::Errno;

/// Implement this function to copy the content of an inode `inode_nr`
/// to a file descriptor `out`. A file at `inode_nr` may be a sparse file.
/// `img` is a file descriptor of an open ext2 image.
///
/// It suffices to support single- and double-indirect blocks.
///
/// If a copy was successful, return `Ok(())`. If an error occurred during
/// a read or a write, return `Err(errno)`.
pub fn dump_file(img: BorrowedFd<'_>, inode_nr: c_int, out: BorrowedFd<'_>) -> Result<(), Errno> {
    let _ = (img, inode_nr, out);
    todo!()
}
