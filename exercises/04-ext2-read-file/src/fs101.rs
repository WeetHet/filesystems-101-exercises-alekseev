#![no_std]

#[path = "../stdlib/fs101_ext2.rs"]
pub mod ext2;

pub use rustix::fd::BorrowedFd;
pub use rustix::io::Errno;

#[doc(hidden)]
#[macro_export]
macro_rules! export_solution {
    ($solution:ident) => {
        const _: () = {
            use ::core::ffi::c_int;
            use $crate::{BorrowedFd, Errno};

            #[unsafe(no_mangle)]
            unsafe extern "C" fn dump_file(img: c_int, inode_nr: c_int, out: c_int) -> c_int {
                let f: fn(BorrowedFd<'_>, c_int, BorrowedFd<'_>) -> Result<(), Errno> =
                    $solution::dump_file;
                let img = unsafe { BorrowedFd::borrow_raw(img) };
                let out = unsafe { BorrowedFd::borrow_raw(out) };
                match f(img, inode_nr, out) {
                    Ok(()) => 0,
                    Err(e) => -e.raw_os_error(),
                }
            }
        };
    };
}
