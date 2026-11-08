#![no_std]

pub use rustix::fd::BorrowedFd;
pub use rustix::io::Errno;

pub mod uring {
    #![allow(non_camel_case_types, non_snake_case, non_upper_case_globals, unsafe_op_in_unsafe_fn)]

    include!(concat!(env!("OUT_DIR"), "/bindings.rs"));
}

#[doc(hidden)]
#[macro_export]
macro_rules! export_solution {
    ($solution:ident) => {
        const _: () = {
            use ::core::ffi::c_int;
            use $crate::{BorrowedFd, Errno};

            #[unsafe(no_mangle)]
            unsafe extern "C" fn copy(fd_in: c_int, fd_out: c_int) -> c_int {
                let f: fn(BorrowedFd<'_>, BorrowedFd<'_>) -> Result<(), Errno> = $solution::copy;
                let fd_in = unsafe { BorrowedFd::borrow_raw(fd_in) };
                let fd_out = unsafe { BorrowedFd::borrow_raw(fd_out) };
                match f(fd_in, fd_out) {
                    Ok(()) => 0,
                    Err(e) => -e.raw_os_error(),
                }
            }
        };
    };
}
