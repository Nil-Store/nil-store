use c_kzg::{
    Blob, Bytes32, Bytes48, KzgCommitment, KzgProof, KzgSettings,
};
pub use c_kzg; // Re-export the crate or types
use std::path::Path;
use thiserror::Error;

#[derive(Error, Debug)]
pub enum KzgError {
    #[error("IO error: {0}")]
    Io(#[from] std::io::Error),
    #[error("KZG error: {0:?}")]
    Internal(c_kzg::Error),
    #[error("Invalid data length")]
    InvalidDataLength,
}

impl From<c_kzg::Error> for KzgError {
    fn from(e: c_kzg::Error) -> Self {
        KzgError::Internal(e)
    }
}

pub struct KzgContext {
    settings: KzgSettings,
}

impl KzgContext {
    pub fn load_from_file<P: AsRef<Path>>(path: P) -> Result<Self, KzgError> {
        let settings = KzgSettings::load_trusted_setup_file(path.as_ref(), 0)
            .map_err(KzgError::Internal)?;
        Ok(Self { settings })
    }

    pub fn blob_to_commitment(&self, blob_bytes: &[u8]) -> Result<KzgCommitment, KzgError> {
        if blob_bytes.len() != c_kzg::BYTES_PER_BLOB {
            return Err(KzgError::InvalidDataLength);
        }
        
        let blob = Blob::from_bytes(blob_bytes).map_err(KzgError::Internal)?;
        self.settings.blob_to_kzg_commitment(&blob)
            .map_err(KzgError::Internal)
    }

    pub fn compute_proof(
        &self,
        blob_bytes: &[u8],
        input_point_bytes: &[u8],
    ) -> Result<(KzgProof, Bytes32), KzgError> {
        if blob_bytes.len() != c_kzg::BYTES_PER_BLOB {
             return Err(KzgError::InvalidDataLength);
        }
        if input_point_bytes.len() != 32 {
             return Err(KzgError::InvalidDataLength);
        }

        let blob = Blob::from_bytes(blob_bytes).map_err(KzgError::Internal)?;
        let z = Bytes32::from_bytes(input_point_bytes).map_err(KzgError::Internal)?;

        self.settings.compute_kzg_proof(&blob, &z)
            .map_err(KzgError::Internal)
    }

    pub fn verify_proof(
        &self,
        commitment_bytes: &[u8],
        input_point_bytes: &[u8],
        claimed_value_bytes: &[u8],
        proof_bytes: &[u8],
    ) -> Result<bool, KzgError> {
         if commitment_bytes.len() != 48 || input_point_bytes.len() != 32 || claimed_value_bytes.len() != 32 || proof_bytes.len() != 48 {
            return Err(KzgError::InvalidDataLength);
        }

        let commitment = Bytes48::from_bytes(commitment_bytes).map_err(KzgError::Internal)?;
        let z = Bytes32::from_bytes(input_point_bytes).map_err(KzgError::Internal)?;
        let y = Bytes32::from_bytes(claimed_value_bytes).map_err(KzgError::Internal)?;
        let proof = Bytes48::from_bytes(proof_bytes).map_err(KzgError::Internal)?;

        self.settings.verify_kzg_proof(
            &commitment,
            &z,
            &y,
            &proof,
        )
        .map_err(KzgError::Internal)
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::path::PathBuf;

    fn get_trusted_setup_path() -> PathBuf {
        let mut path = PathBuf::from(env!("CARGO_MANIFEST_DIR"));
        path.pop(); // Go up to root
        path.push("demos");
        path.push("kzg");
        path.push("trusted_setup.txt");
        path
    }

    #[test]
    fn test_load_trusted_setup() {
        let path = get_trusted_setup_path();
        assert!(path.exists(), "Trusted setup file not found at {:?}", path);
        let ctx = KzgContext::load_from_file(&path);
        assert!(ctx.is_ok());
    }

    #[test]
    fn test_commit_prove_verify() {
        let path = get_trusted_setup_path();
        let ctx = KzgContext::load_from_file(&path).unwrap();

        // Create a dummy blob (all zeros except first byte)
        let mut blob_bytes = [0u8; c_kzg::BYTES_PER_BLOB];
        blob_bytes[0] = 1; // Just some data

        // Commit
        let commitment = ctx.blob_to_commitment(&blob_bytes).expect("Commit failed");
        
        // Point to evaluate at (z)
        let mut z_bytes = [0u8; 32];
        z_bytes[0] = 2; // Just some point

        // Compute proof
        let (proof, y) = ctx.compute_proof(&blob_bytes, &z_bytes).expect("Proof failed");

        // Verify
        let valid = ctx.verify_proof(
            commitment.as_slice(),
            &z_bytes,
            y.as_slice(),
            proof.as_slice()
        ).expect("Verification failed");

        assert!(valid, "Proof should be valid");
    }
}
