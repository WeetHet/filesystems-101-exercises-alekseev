#![no_std]

use core::ffi::{CStr, c_char, c_int};

#[path = "../stdlib/fs101_ext2.rs"]
pub mod ext2;

pub use rustix::fd::BorrowedFd;
pub use rustix::io::Errno;

pub mod ctest {
    use core::ffi::{c_char, c_int};

    unsafe extern "C" {
        pub fn report_file(inode_nr: c_int, type_: c_char, name: *const c_char);
    }
}

pub fn report_file(inode_nr: c_int, file_type: u8, name: &CStr) {
    unsafe { ctest::report_file(inode_nr, file_type as c_char, name.as_ptr()) }
}

#[doc(hidden)]
#[macro_export]
macro_rules! export_solution {
    ($solution:ident) => {
        const _: () = {
            use ::core::ffi::c_int;
            use $crate::{BorrowedFd, Errno};

            #[unsafe(no_mangle)]
            unsafe extern "C" fn dump_dir(img: c_int, inode_nr: c_int) -> c_int {
                let f: fn(BorrowedFd<'_>, c_int) -> Result<(), Errno> = $solution::dump_dir;
                let img = unsafe { BorrowedFd::borrow_raw(img) };
                match f(img, inode_nr) {
                    Ok(()) => 0,
                    Err(e) => -e.raw_os_error(),
                }
            }
        };
    };
}
