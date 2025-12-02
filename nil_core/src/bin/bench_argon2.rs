use argon2::{
    password_hash::{
        PasswordHasher, SaltString, Error as HashError
    },
    Argon2, Params, Algorithm, Version
};
use std::time::Instant;

pub fn main() {
    println!("Running Argon2id benchmark...");

    // Parameters from AGENTS.md / nil_core default:
    // Memory: 15 MiB (15 * 1024 KB)
    // Time: 2 passes
    // Parallelism: 1 lane
    let params = Params::new(
        15 * 1024,
        2,
        1,
        None
    ).expect("Invalid params");

    let argon2 = Argon2::new(Algorithm::Argon2id, Version::V0x13, params);
    let salt = SaltString::generate(&mut argon2::password_hash::rand_core::OsRng);
    let password = b"some_random_data_block_for_sealing";

    println!("Params: m=15MB, t=2, p=1");
    
    let iterations = 10;
    let mut total_time = std::time::Duration::new(0, 0);

    for i in 0..iterations {
        let start = Instant::now();
        let _ = argon2.hash_password(password, &salt).expect("Hash failed");
        let duration = start.elapsed();
        println!("Iter {}: {:?}", i, duration);
        total_time += duration;
    }

    println!("Average time: {:?}", total_time / iterations);
}
