use anyhow::{Context, Result};
use clap::{Parser, Subcommand};
use nil_core::{
    kzg::KzgContext,
    utils::{
        bytes_to_fr_be, file_to_symbols, frs_to_blobs, sha256_to_fr,
        symbols_to_frs, z_for_cell, SYMBOLS_PER_BLOB,
    },
};
use rand::{Rng, SeedableRng};
use rand::rngs::StdRng;
use serde::{Deserialize, Serialize};
use sha2::{Digest, Sha256};
use std::path::PathBuf;

#[derive(Parser)]
#[command(name = "nil-cli")]
struct Cli {
    #[command(subcommand)]
    command: Commands,

    #[arg(long, env = "CKZG_TRUSTED_SETUP", default_value = "trusted_setup.txt")]
    trusted_setup: PathBuf,
}

#[derive(Subcommand)]
enum Commands {
    Shard {
        file: PathBuf,
        #[arg(long, default_value = "5,17,42")]
        seeds: String,
        #[arg(long, default_value = "output.json")]
        out: PathBuf,
    },
    Verify {
        file: PathBuf,
    },
}

#[derive(Serialize, Deserialize)]
struct Output {
    filename: String,
    file_size_bytes: u64,
    symbols_1kib: usize,
    blob_count: usize,
    du_c_root_hex: String,
    commitments_hex: Vec<String>,
    proofs: Vec<ProofData>,
}

#[derive(Serialize, Deserialize)]
struct ProofData {
    index: usize,
    shard_idx: usize,
    local_cell: usize,
    z_hex: String,
    y_hex: String,
    commitment: String,
    symbol_preview_hex: String,
    proof_hex: String,
    verified: bool,
}

fn main() -> Result<()> {
    let cli = Cli::parse();

    if !cli.trusted_setup.exists() {
        let fallback = PathBuf::from("demos/kzg/trusted_setup.txt");
        if fallback.exists() && cli.trusted_setup.to_string_lossy() == "trusted_setup.txt" {
            eprintln!("Using fallback trusted setup at {:?}", fallback);
            return run(cli.command, fallback);
        }
    }

    run(cli.command, cli.trusted_setup)
}

fn run(cmd: Commands, ts_path: PathBuf) -> Result<()> {
    let kzg_ctx = KzgContext::load_from_file(&ts_path)
        .context("Failed to load KZG trusted setup")?;

    match cmd {
        Commands::Shard { file, seeds, out } => {
            println!("Sharding file: {:?}", file);
            let data = std::fs::read(&file).context("Failed to read input file")?;
            let symbols = file_to_symbols(&data);
            let ys = symbols_to_frs(&symbols);
            let blobs = frs_to_blobs(&ys);

            let mut shards = Vec::new();
            let mut commitments = Vec::new();
            let mut commitments_hex = Vec::new();

            println!("Committing to {} blobs...", blobs.len());
            for (i, blob) in blobs.iter().enumerate() {
                let commitment = kzg_ctx.blob_to_commitment(blob)
                    .context(format!("Failed to commit blob {}", i))?;
                let commitment_bytes = commitment.to_bytes();
                commitments.push(commitment_bytes);
                commitments_hex.push(format!("0x{}", hex::encode(commitment_bytes.as_slice())));
                shards.push((i * SYMBOLS_PER_BLOB, blob, commitment_bytes));
            }

            // DU Root
            let mut hasher = Sha256::new();
            hasher.update(b"NIL_DEMO_C_ROOT");
            for c in &commitments {
                hasher.update(c.as_slice());
            }
            let du_root = hasher.finalize();
            let du_root_hex = format!("0x{}", hex::encode(du_root));

            let seed_list: Vec<u64> = seeds.split(',')
                .filter_map(|s| s.trim().parse().ok())
                .collect();

            let mut proofs = Vec::new();

            for seed in seed_list {
                let mut rng = StdRng::seed_from_u64(seed);
                let index = rng.random_range(0..symbols.len());
                
                let shard_idx = index / SYMBOLS_PER_BLOB;
                let local_cell = index % SYMBOLS_PER_BLOB;

                if shard_idx >= shards.len() {
                    continue;
                }
                let (_, blob, commitment) = &shards[shard_idx];

                let z_bytes = z_for_cell(local_cell);
                let (proof, y_bytes) = kzg_ctx.compute_proof(blob, &z_bytes)?;
                let proof_bytes = proof.to_bytes();

                // Verify locally
                let valid = kzg_ctx.verify_proof(
                    commitment.as_slice(), 
                    &z_bytes, 
                    y_bytes.as_slice(), 
                    proof_bytes.as_slice()
                )?;

                let symbol = &symbols[index];
                let computed_fr = sha256_to_fr(symbol);
                // y_bytes should be BE now
                let computed_y = bytes_to_fr_be(y_bytes.as_slice());
                
                let binding_valid = computed_fr == computed_y;

                proofs.push(ProofData {
                    index,
                    shard_idx,
                    local_cell,
                    z_hex: format!("0x{}", hex::encode(z_bytes)),
                    y_hex: format!("0x{}", hex::encode(y_bytes.as_slice())),
                    commitment: format!("0x{}", hex::encode(commitment.as_slice())),
                    symbol_preview_hex: hex::encode(&symbol[0..std::cmp::min(16, symbol.len())]),
                    proof_hex: format!("0x{}", hex::encode(proof_bytes.as_slice())),
                    verified: valid && binding_valid,
                });
            }

            let output = Output {
                filename: file.to_string_lossy().into_owned(),
                file_size_bytes: data.len() as u64,
                symbols_1kib: symbols.len(),
                blob_count: blobs.len(),
                du_c_root_hex: du_root_hex,
                commitments_hex,
                proofs,
            };

            let json = serde_json::to_string_pretty(&output)?;
            std::fs::write(&out, json)?;
            println!("Saved output to {:?}", out);
        }
        Commands::Verify { file } => {
            println!("Verifying proofs in: {:?}", file);
            let data = std::fs::read_to_string(&file).context("Failed to read proof file")?;
            let output: Output = serde_json::from_str(&data)?;

            let original_file = PathBuf::from(&output.filename);
            let original_data = if original_file.exists() {
                Some(std::fs::read(&original_file)?)
            } else {
                println!("Warning: Original file {:?} not found. Only verifying KZG proofs, not data binding.", original_file);
                None
            };

            let mut all_valid = true;

            for (i, proof) in output.proofs.iter().enumerate() {
                print!("Proof {} (idx {}): ", i, proof.index);
                
                let z = hex::decode(&proof.z_hex[2..])?;
                let y = hex::decode(&proof.y_hex[2..])?;
                let commitment = hex::decode(&proof.commitment[2..])?;
                let proof_bytes = hex::decode(&proof.proof_hex[2..])?;

                let kzg_valid = kzg_ctx.verify_proof(&commitment, &z, &y, &proof_bytes)?;
                
                if !kzg_valid {
                    println!("KZG FAIL");
                    all_valid = false;
                    continue;
                }

                if let Some(ref data) = original_data {
                    let symbols = file_to_symbols(data);
                    if proof.index < symbols.len() {
                        let symbol = &symbols[proof.index];
                        let fr = sha256_to_fr(symbol);
                        let y_fr = bytes_to_fr_be(&y);
                        if fr != y_fr {
                            println!("Binding FAIL (Data mismatch)");
                            all_valid = false;
                            continue;
                        }
                        println!("OK (Full)");
                    } else {
                         println!("Index out of bounds of original file");
                         all_valid = false;
                    }
                } else {
                    println!("OK (KZG only)");
                }
            }

            if all_valid {
                println!("All proofs valid.");
            } else {
                println!("Some proofs failed.");
                std::process::exit(1);
            }
        }
    }
    Ok(())
}
