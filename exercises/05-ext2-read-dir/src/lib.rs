#![no_std]

use core::ffi::c_int;

use rustix::fd::BorrowedFd;
use rustix::io::Errno;

/// Implement this function to parse the content of an inode `inode_nr`
/// as an ext2 directory. The function must call fs101::report_file() for each
/// directory entry it finds.
///
/// Directory entries are guaranteed to be regular files or directories.
///
/// If all directory entries were read and parsed successfully, return `Ok(())`.
/// Otherwise, return `Err(errno)`.
pub fn dump_dir(img: BorrowedFd<'_>, inode_nr: c_int) -> Result<(), Errno> {
    let _ = (img, inode_nr);
    todo!()
}
