#![no_std]

#[path = "../stdlib/fs101_fuse.rs"]
pub mod fuse;

#[path = "../stdlib/fs101_ext2.rs"]
pub mod ext2;

pub use rustix::fd::BorrowedFd;

#[doc(hidden)]
#[macro_export]
macro_rules! export_solution {
    ($solution:ident) => {
        const _: () = {
            use ::core::ffi::{CStr, c_char, c_int};
            use $crate::BorrowedFd;

            #[unsafe(no_mangle)]
            unsafe extern "C" fn ext2fuse(img: c_int, mntp: *const c_char) -> c_int {
                let f: fn(BorrowedFd<'_>, &CStr) -> c_int = $solution::ext2fuse;
                let img = unsafe { BorrowedFd::borrow_raw(img) };
                f(img, unsafe { CStr::from_ptr(mntp) })
            }
        };
    };
}
