#![allow(non_camel_case_types)]

use core::ffi::{CStr, c_char, c_int, c_uint, c_void};
use core::marker::PhantomData;
use core::{mem, ptr, slice};

use libc::{dev_t, flock, gid_t, mode_t, off_t, pid_t, size_t, ssize_t, stat, statvfs, timespec, uid_t};
use rustix::io::Errno;

#[repr(C)]
#[derive(Debug)]
pub struct fuse_file_info {
    pub flags: c_int,
    pub bits: c_uint,
    padding2: c_uint,
    pub fh: u64,
    pub lock_owner: u64,
    pub poll_events: u32,
}

macro_rules! file_info_bits {
    ($($get:ident, $set:ident = $bit:expr;)*) => {
        impl fuse_file_info {
            $(
                pub fn $get(&self) -> bool {
                    self.bits & (1 << $bit) != 0
                }

                pub fn $set(&mut self, on: bool) {
                    if on {
                        self.bits |= 1 << $bit;
                    } else {
                        self.bits &= !(1 << $bit);
                    }
                }
            )*
        }
    };
}

file_info_bits! {
    writepage, set_writepage = 0;
    direct_io, set_direct_io = 1;
    keep_cache, set_keep_cache = 2;
    flush, set_flush = 3;
    nonseekable, set_nonseekable = 4;
    flock_release, set_flock_release = 5;
    cache_readdir, set_cache_readdir = 6;
    noflush, set_noflush = 7;
}

#[repr(C)]
pub struct fuse_conn_info {
    _private: [u8; 0],
}

#[repr(C)]
pub struct fuse_config {
    _private: [u8; 0],
}

#[repr(C)]
pub struct fuse_pollhandle {
    _private: [u8; 0],
}

#[repr(C)]
pub struct fuse_bufvec {
    _private: [u8; 0],
}

#[repr(C)]
pub struct fuse {
    _private: [u8; 0],
}

pub type fuse_readdir_flags = c_uint;
pub const FUSE_READDIR_PLUS: fuse_readdir_flags = 1 << 0;

pub type fuse_fill_dir_flags = c_uint;
pub const FUSE_FILL_DIR_PLUS: fuse_fill_dir_flags = 1 << 1;

pub type fuse_fill_dir_t = Option<
    unsafe extern "C" fn(
        buf: *mut c_void,
        name: *const c_char,
        stbuf: *const stat,
        off: off_t,
        flags: fuse_fill_dir_flags,
    ) -> c_int,
>;

#[repr(C)]
#[derive(Clone, Copy)]
pub struct fuse_operations {
    pub getattr: Option<unsafe extern "C" fn(*const c_char, *mut stat, *mut fuse_file_info) -> c_int>,
    pub readlink: Option<unsafe extern "C" fn(*const c_char, *mut c_char, size_t) -> c_int>,
    pub mknod: Option<unsafe extern "C" fn(*const c_char, mode_t, dev_t) -> c_int>,
    pub mkdir: Option<unsafe extern "C" fn(*const c_char, mode_t) -> c_int>,
    pub unlink: Option<unsafe extern "C" fn(*const c_char) -> c_int>,
    pub rmdir: Option<unsafe extern "C" fn(*const c_char) -> c_int>,
    pub symlink: Option<unsafe extern "C" fn(*const c_char, *const c_char) -> c_int>,
    pub rename: Option<unsafe extern "C" fn(*const c_char, *const c_char, c_uint) -> c_int>,
    pub link: Option<unsafe extern "C" fn(*const c_char, *const c_char) -> c_int>,
    pub chmod: Option<unsafe extern "C" fn(*const c_char, mode_t, *mut fuse_file_info) -> c_int>,
    pub chown: Option<unsafe extern "C" fn(*const c_char, uid_t, gid_t, *mut fuse_file_info) -> c_int>,
    pub truncate: Option<unsafe extern "C" fn(*const c_char, off_t, *mut fuse_file_info) -> c_int>,
    pub open: Option<unsafe extern "C" fn(*const c_char, *mut fuse_file_info) -> c_int>,
    pub read: Option<
        unsafe extern "C" fn(*const c_char, *mut c_char, size_t, off_t, *mut fuse_file_info) -> c_int,
    >,
    pub write: Option<
        unsafe extern "C" fn(*const c_char, *const c_char, size_t, off_t, *mut fuse_file_info) -> c_int,
    >,
    pub statfs: Option<unsafe extern "C" fn(*const c_char, *mut statvfs) -> c_int>,
    pub flush: Option<unsafe extern "C" fn(*const c_char, *mut fuse_file_info) -> c_int>,
    pub release: Option<unsafe extern "C" fn(*const c_char, *mut fuse_file_info) -> c_int>,
    pub fsync: Option<unsafe extern "C" fn(*const c_char, c_int, *mut fuse_file_info) -> c_int>,
    pub setxattr: Option<
        unsafe extern "C" fn(*const c_char, *const c_char, *const c_char, size_t, c_int) -> c_int,
    >,
    pub getxattr: Option<unsafe extern "C" fn(*const c_char, *const c_char, *mut c_char, size_t) -> c_int>,
    pub listxattr: Option<unsafe extern "C" fn(*const c_char, *mut c_char, size_t) -> c_int>,
    pub removexattr: Option<unsafe extern "C" fn(*const c_char, *const c_char) -> c_int>,
    pub opendir: Option<unsafe extern "C" fn(*const c_char, *mut fuse_file_info) -> c_int>,
    pub readdir: Option<
        unsafe extern "C" fn(
            *const c_char,
            *mut c_void,
            fuse_fill_dir_t,
            off_t,
            *mut fuse_file_info,
            fuse_readdir_flags,
        ) -> c_int,
    >,
    pub releasedir: Option<unsafe extern "C" fn(*const c_char, *mut fuse_file_info) -> c_int>,
    pub fsyncdir: Option<unsafe extern "C" fn(*const c_char, c_int, *mut fuse_file_info) -> c_int>,
    pub init: Option<unsafe extern "C" fn(*mut fuse_conn_info, *mut fuse_config) -> *mut c_void>,
    pub destroy: Option<unsafe extern "C" fn(*mut c_void)>,
    pub access: Option<unsafe extern "C" fn(*const c_char, c_int) -> c_int>,
    pub create: Option<unsafe extern "C" fn(*const c_char, mode_t, *mut fuse_file_info) -> c_int>,
    pub lock: Option<unsafe extern "C" fn(*const c_char, *mut fuse_file_info, c_int, *mut flock) -> c_int>,
    pub utimens: Option<unsafe extern "C" fn(*const c_char, *const timespec, *mut fuse_file_info) -> c_int>,
    pub bmap: Option<unsafe extern "C" fn(*const c_char, size_t, *mut u64) -> c_int>,
    pub ioctl: Option<
        unsafe extern "C" fn(*const c_char, c_int, *mut c_void, *mut fuse_file_info, c_uint, *mut c_void) -> c_int,
    >,
    pub poll: Option<
        unsafe extern "C" fn(*const c_char, *mut fuse_file_info, *mut fuse_pollhandle, *mut c_uint) -> c_int,
    >,
    pub write_buf: Option<
        unsafe extern "C" fn(*const c_char, *mut fuse_bufvec, off_t, *mut fuse_file_info) -> c_int,
    >,
    pub read_buf: Option<
        unsafe extern "C" fn(*const c_char, *mut *mut fuse_bufvec, size_t, off_t, *mut fuse_file_info) -> c_int,
    >,
    pub flock: Option<unsafe extern "C" fn(*const c_char, *mut fuse_file_info, c_int) -> c_int>,
    pub fallocate: Option<unsafe extern "C" fn(*const c_char, c_int, off_t, off_t, *mut fuse_file_info) -> c_int>,
    pub copy_file_range: Option<
        unsafe extern "C" fn(
            *const c_char,
            *mut fuse_file_info,
            off_t,
            *const c_char,
            *mut fuse_file_info,
            off_t,
            size_t,
            c_int,
        ) -> ssize_t,
    >,
    pub lseek: Option<unsafe extern "C" fn(*const c_char, off_t, c_int, *mut fuse_file_info) -> off_t>,
}

impl fuse_operations {
    pub const EMPTY: Self = Self {
        getattr: None,
        readlink: None,
        mknod: None,
        mkdir: None,
        unlink: None,
        rmdir: None,
        symlink: None,
        rename: None,
        link: None,
        chmod: None,
        chown: None,
        truncate: None,
        open: None,
        read: None,
        write: None,
        statfs: None,
        flush: None,
        release: None,
        fsync: None,
        setxattr: None,
        getxattr: None,
        listxattr: None,
        removexattr: None,
        opendir: None,
        readdir: None,
        releasedir: None,
        fsyncdir: None,
        init: None,
        destroy: None,
        access: None,
        create: None,
        lock: None,
        utimens: None,
        bmap: None,
        ioctl: None,
        poll: None,
        write_buf: None,
        read_buf: None,
        flock: None,
        fallocate: None,
        copy_file_range: None,
        lseek: None,
    };
}

impl Default for fuse_operations {
    fn default() -> Self {
        Self::EMPTY
    }
}

#[repr(C)]
#[derive(Debug)]
pub struct fuse_context {
    pub fuse: *mut fuse,
    pub uid: uid_t,
    pub gid: gid_t,
    pub pid: pid_t,
    pub private_data: *mut c_void,
    pub umask: mode_t,
}

unsafe extern "C" {
    pub fn fuse_get_context() -> *mut fuse_context;

    pub fn fuse_main_real(
        argc: c_int,
        argv: *mut *mut c_char,
        op: *const fuse_operations,
        op_size: size_t,
        private_data: *mut c_void,
    ) -> c_int;
}

pub fn fuse_main(args: &[&CStr], op: &fuse_operations, private_data: *mut c_void) -> c_int {
    const MAX_ARGS: usize = 32;
    assert!(args.len() < MAX_ARGS, "fuse_main: too many arguments");

    let mut argv = [ptr::null_mut::<c_char>(); MAX_ARGS];
    for (slot, arg) in argv.iter_mut().zip(args) {
        *slot = arg.as_ptr().cast_mut();
    }
    unsafe {
        fuse_main_real(
            args.len() as c_int,
            argv.as_mut_ptr(),
            op,
            size_of::<fuse_operations>(),
            private_data,
        )
    }
}

#[derive(Clone, Copy, Debug)]
pub struct Context {
    pub uid: uid_t,
    pub gid: gid_t,
    pub pid: pid_t,
    pub umask: mode_t,
}

pub fn context() -> Option<Context> {
    let ctx = unsafe { fuse_get_context().as_ref() }?;
    Some(Context {
        uid: ctx.uid,
        gid: ctx.gid,
        pid: ctx.pid,
        umask: ctx.umask,
    })
}

pub struct DirFiller<'a> {
    buf: *mut c_void,
    filler: unsafe extern "C" fn(*mut c_void, *const c_char, *const stat, off_t, fuse_fill_dir_flags) -> c_int,
    _marker: PhantomData<&'a mut c_void>,
}

impl DirFiller<'_> {
    pub fn add(&mut self, name: &CStr, mode: mode_t) -> bool {
        let mut st: stat = unsafe { mem::zeroed() };
        st.st_mode = mode;
        unsafe { (self.filler)(self.buf, name.as_ptr(), &st, 0, 0) == 0 }
    }
}

#[allow(unused_variables)]
pub trait Filesystem: Sync {
    fn getattr(&self, path: &CStr, st: &mut stat) -> Result<(), Errno> {
        Err(Errno::NOSYS)
    }

    fn readdir(&self, path: &CStr, filler: &mut DirFiller<'_>) -> Result<(), Errno> {
        Err(Errno::NOSYS)
    }

    fn open(&self, path: &CStr, fi: &mut fuse_file_info) -> Result<(), Errno> {
        Ok(())
    }

    fn read(&self, path: &CStr, buf: &mut [u8], offset: u64, fi: &mut fuse_file_info) -> Result<usize, Errno> {
        Err(Errno::NOSYS)
    }

    fn write(&self, path: &CStr, buf: &[u8], offset: u64, fi: &mut fuse_file_info) -> Result<usize, Errno> {
        Err(Errno::NOSYS)
    }

    fn create(&self, path: &CStr, mode: mode_t, fi: &mut fuse_file_info) -> Result<(), Errno> {
        Err(Errno::NOSYS)
    }

    fn mknod(&self, path: &CStr, mode: mode_t, dev: dev_t) -> Result<(), Errno> {
        Err(Errno::NOSYS)
    }

    fn mkdir(&self, path: &CStr, mode: mode_t) -> Result<(), Errno> {
        Err(Errno::NOSYS)
    }

    fn unlink(&self, path: &CStr) -> Result<(), Errno> {
        Err(Errno::NOSYS)
    }

    fn rmdir(&self, path: &CStr) -> Result<(), Errno> {
        Err(Errno::NOSYS)
    }

    fn symlink(&self, target: &CStr, path: &CStr) -> Result<(), Errno> {
        Err(Errno::NOSYS)
    }

    fn rename(&self, from: &CStr, to: &CStr, flags: c_uint) -> Result<(), Errno> {
        Err(Errno::NOSYS)
    }

    fn link(&self, from: &CStr, to: &CStr) -> Result<(), Errno> {
        Err(Errno::NOSYS)
    }

    fn chmod(&self, path: &CStr, mode: mode_t) -> Result<(), Errno> {
        Err(Errno::NOSYS)
    }

    fn chown(&self, path: &CStr, uid: uid_t, gid: gid_t) -> Result<(), Errno> {
        Err(Errno::NOSYS)
    }

    fn truncate(&self, path: &CStr, size: u64) -> Result<(), Errno> {
        Err(Errno::NOSYS)
    }
}

unsafe fn this<'a, F: Filesystem>() -> &'a F {
    unsafe { &*(*fuse_get_context()).private_data.cast::<F>() }
}

unsafe fn cstr<'a>(s: *const c_char) -> &'a CStr {
    unsafe { CStr::from_ptr(s) }
}

fn status(r: Result<(), Errno>) -> c_int {
    match r {
        Ok(()) => 0,
        Err(e) => -e.raw_os_error(),
    }
}

fn count(r: Result<usize, Errno>) -> c_int {
    match r {
        Ok(n) => n.min(c_int::MAX as usize) as c_int,
        Err(e) => -e.raw_os_error(),
    }
}

unsafe extern "C" fn op_getattr<F: Filesystem>(path: *const c_char, st: *mut stat, _fi: *mut fuse_file_info) -> c_int {
    unsafe { status(this::<F>().getattr(cstr(path), &mut *st)) }
}

unsafe extern "C" fn op_readdir<F: Filesystem>(
    path: *const c_char,
    buf: *mut c_void,
    filler: fuse_fill_dir_t,
    _off: off_t,
    _fi: *mut fuse_file_info,
    _flags: fuse_readdir_flags,
) -> c_int {
    let Some(filler) = filler else { return -libc::EIO };
    let mut filler = DirFiller {
        buf,
        filler,
        _marker: PhantomData,
    };
    unsafe { status(this::<F>().readdir(cstr(path), &mut filler)) }
}

unsafe extern "C" fn op_open<F: Filesystem>(path: *const c_char, fi: *mut fuse_file_info) -> c_int {
    unsafe { status(this::<F>().open(cstr(path), &mut *fi)) }
}

unsafe extern "C" fn op_read<F: Filesystem>(
    path: *const c_char,
    buf: *mut c_char,
    size: size_t,
    off: off_t,
    fi: *mut fuse_file_info,
) -> c_int {
    unsafe {
        let buf = slice::from_raw_parts_mut(buf.cast::<u8>(), size);
        count(this::<F>().read(cstr(path), buf, off as u64, &mut *fi))
    }
}

unsafe extern "C" fn op_write<F: Filesystem>(
    path: *const c_char,
    buf: *const c_char,
    size: size_t,
    off: off_t,
    fi: *mut fuse_file_info,
) -> c_int {
    unsafe {
        let buf = slice::from_raw_parts(buf.cast::<u8>(), size);
        count(this::<F>().write(cstr(path), buf, off as u64, &mut *fi))
    }
}

unsafe extern "C" fn op_create<F: Filesystem>(path: *const c_char, mode: mode_t, fi: *mut fuse_file_info) -> c_int {
    unsafe { status(this::<F>().create(cstr(path), mode, &mut *fi)) }
}

unsafe extern "C" fn op_mknod<F: Filesystem>(path: *const c_char, mode: mode_t, dev: dev_t) -> c_int {
    unsafe { status(this::<F>().mknod(cstr(path), mode, dev)) }
}

unsafe extern "C" fn op_mkdir<F: Filesystem>(path: *const c_char, mode: mode_t) -> c_int {
    unsafe { status(this::<F>().mkdir(cstr(path), mode)) }
}

unsafe extern "C" fn op_unlink<F: Filesystem>(path: *const c_char) -> c_int {
    unsafe { status(this::<F>().unlink(cstr(path))) }
}

unsafe extern "C" fn op_rmdir<F: Filesystem>(path: *const c_char) -> c_int {
    unsafe { status(this::<F>().rmdir(cstr(path))) }
}

unsafe extern "C" fn op_symlink<F: Filesystem>(target: *const c_char, path: *const c_char) -> c_int {
    unsafe { status(this::<F>().symlink(cstr(target), cstr(path))) }
}

unsafe extern "C" fn op_rename<F: Filesystem>(from: *const c_char, to: *const c_char, flags: c_uint) -> c_int {
    unsafe { status(this::<F>().rename(cstr(from), cstr(to), flags)) }
}

unsafe extern "C" fn op_link<F: Filesystem>(from: *const c_char, to: *const c_char) -> c_int {
    unsafe { status(this::<F>().link(cstr(from), cstr(to))) }
}

unsafe extern "C" fn op_chmod<F: Filesystem>(path: *const c_char, mode: mode_t, _fi: *mut fuse_file_info) -> c_int {
    unsafe { status(this::<F>().chmod(cstr(path), mode)) }
}

unsafe extern "C" fn op_chown<F: Filesystem>(
    path: *const c_char,
    uid: uid_t,
    gid: gid_t,
    _fi: *mut fuse_file_info,
) -> c_int {
    unsafe { status(this::<F>().chown(cstr(path), uid, gid)) }
}

unsafe extern "C" fn op_truncate<F: Filesystem>(path: *const c_char, size: off_t, _fi: *mut fuse_file_info) -> c_int {
    unsafe { status(this::<F>().truncate(cstr(path), size as u64)) }
}

pub fn main<F: Filesystem>(args: &[&CStr], fs: &F) -> c_int {
    let ops = fuse_operations {
        getattr: Some(op_getattr::<F>),
        readdir: Some(op_readdir::<F>),
        open: Some(op_open::<F>),
        read: Some(op_read::<F>),
        write: Some(op_write::<F>),
        create: Some(op_create::<F>),
        mknod: Some(op_mknod::<F>),
        mkdir: Some(op_mkdir::<F>),
        unlink: Some(op_unlink::<F>),
        rmdir: Some(op_rmdir::<F>),
        symlink: Some(op_symlink::<F>),
        rename: Some(op_rename::<F>),
        link: Some(op_link::<F>),
        chmod: Some(op_chmod::<F>),
        chown: Some(op_chown::<F>),
        truncate: Some(op_truncate::<F>),
        ..fuse_operations::EMPTY
    };
    fuse_main(args, &ops, ptr::from_ref(fs).cast_mut().cast())
}
