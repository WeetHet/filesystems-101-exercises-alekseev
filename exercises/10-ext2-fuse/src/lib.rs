#![no_std]

use core::ffi::{CStr, c_int};

use rustix::fd::BorrowedFd;

/// Implement this function to mount an ext2 file system image `img`
/// via FUSE to `mntp`.
///
/// Any attempt to write to the FS must report EROFS.
pub fn ext2fuse(img: BorrowedFd<'_>, mntp: &CStr) -> c_int {
    let _ = (img, mntp);
    todo!()
}
