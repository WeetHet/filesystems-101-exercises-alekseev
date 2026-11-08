#![allow(non_camel_case_types, non_snake_case, non_upper_case_globals, unsafe_op_in_unsafe_fn)]

use bytemuck::{Pod, Zeroable};

include!(concat!(env!("OUT_DIR"), "/bindings.rs"));

const _: () = assert!(size_of::<ext2_super_block>() == 1024);
const _: () = assert!(size_of::<ext2_group_desc>() == 32);
const _: () = assert!(size_of::<ext2_inode>() == 128);

unsafe impl Zeroable for ext2_super_block {}
unsafe impl Pod for ext2_super_block {}
unsafe impl Zeroable for ext2_group_desc {}
unsafe impl Pod for ext2_group_desc {}
unsafe impl Zeroable for ext2_inode {}
unsafe impl Pod for ext2_inode {}
