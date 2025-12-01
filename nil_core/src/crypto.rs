use argon2::{
    password_hash::{
        PasswordHasher, SaltString, Error as HashError
    },
    Argon2, Params, Algorithm, Version
};
use thiserror::Error;

#[derive(Error, Debug)]
pub enum CryptoError {
    #[error("Argon2 error: {0}")]
    Argon2(#[from] HashError),
    #[error("Argon2 params error: {0}")]
    Params(#[from] argon2::Error),
}

pub struct Argon2Context {
    params: Params,
}

impl Default for Argon2Context {
    fn default() -> Self {
        // Default parameters for Nilcoin "Seal"
        let params = Params::new(
            15 * 1024, // Memory cost: 15 MiB
            2,         // Time cost: 2 passes
            1,         // Parallelism: 1 lane
            None
        ).unwrap();
        
        Self { params }
    }
}

impl Argon2Context {
    pub fn new(memory_cost_kb: u32, time_cost: u32, parallelism: u32) -> Result<Self, CryptoError> {
         let params = Params::new(
            memory_cost_kb,
            time_cost,
            parallelism,
            None
        ).map_err(CryptoError::Params)?;
        Ok(Self { params })
    }

    pub fn hash(&self, data: &[u8], salt: &[u8]) -> Result<Vec<u8>, CryptoError> {
        let argon2 = Argon2::new(Algorithm::Argon2id, Version::V0x13, self.params.clone());
        
        let salt_string = SaltString::encode_b64(salt).map_err(CryptoError::Argon2)?;
        let password_hash = argon2.hash_password(data, &salt_string).map_err(CryptoError::Argon2)?;
        
        // Extract the raw hash (output)
        if let Some(output) = password_hash.hash {
             Ok(output.as_bytes().to_vec())
        } else {
             Err(CryptoError::Argon2(HashError::Crypto)) 
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_argon2_hash() {
        let ctx = Argon2Context::default();
        let data = b"hello world";
        let salt = b"somesalt12345678"; // Must be long enough?
        
        let hash = ctx.hash(data, salt);
        assert!(hash.is_ok());
        let hash_bytes = hash.unwrap();
        assert!(!hash_bytes.is_empty());
        println!("Argon2 hash: {}", hex::encode(hash_bytes));
    }
}
