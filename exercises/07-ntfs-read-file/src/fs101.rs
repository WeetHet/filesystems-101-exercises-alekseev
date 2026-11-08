#![no_std]

pub use rustix::fd::BorrowedFd;
pub use rustix::io::Errno;

pub mod ntfs {
    #![allow(non_camel_case_types, non_snake_case, non_upper_case_globals, unsafe_op_in_unsafe_fn)]

    include!(concat!(env!("OUT_DIR"), "/bindings.rs"));
}

pub fn errno() -> Errno {
    Errno::from_raw_os_error(unsafe { *libc::__errno_location() })
}

#[doc(hidden)]
#[macro_export]
macro_rules! export_solution {
    ($solution:ident) => {
        const _: () = {
            use ::core::ffi::{CStr, c_char, c_int};
            use $crate::{BorrowedFd, Errno};

            #[unsafe(no_mangle)]
            unsafe extern "C" fn dump_file(img: c_int, path: *const c_char, out: c_int) -> c_int {
                let f: fn(BorrowedFd<'_>, &CStr, BorrowedFd<'_>) -> Result<(), Errno> =
                    $solution::dump_file;
                let img = unsafe { BorrowedFd::borrow_raw(img) };
                let out = unsafe { BorrowedFd::borrow_raw(out) };
                match f(img, unsafe { CStr::from_ptr(path) }, out) {
                    Ok(()) => 0,
                    Err(e) => -e.raw_os_error(),
                }
            }
        };
    };
}
