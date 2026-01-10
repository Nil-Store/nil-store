use anyhow::Result;
use axum::{routing::post, Json, Router};
use clap::Parser;
use nil_p2p::{AskForProxy, Command, NilNode, ProxyOffer};
use std::sync::atomic::{AtomicU64, Ordering};
use std::sync::Arc;
use std::time::{Duration, SystemTime, UNIX_EPOCH};
use tokio::sync::{mpsc, oneshot};
use tracing::info;
use tracing_subscriber::EnvFilter;

#[derive(Parser, Debug)]
#[command(name = "nil-p2p")]
struct Cli {
    #[arg(long, default_value = "0")]
    port: u16,

    #[arg(long, default_value = "0")]
    seed: u64,

    #[arg(long, default_value = "")]
    proxy_endpoint: String,

    #[arg(long, default_value = "0")]
    proxy_price: u64,

    #[arg(long, default_value = "")]
    http_addr: String,

    #[arg(long, default_value = "5000")]
    proxy_timeout_ms: u64,
}

#[derive(Debug, serde::Deserialize)]
struct ProxyAskRequest {
    deal_id: u64,
    provider: String,
    file_path: String,
    range_start: u64,
    range_len: u64,
    max_price: u64,
}

#[derive(Debug, serde::Serialize)]
struct ProxyAskResponse {
    request_id: String,
    deputy_peer_id: String,
    deputy_endpoint: String,
    price: u64,
}

struct AppState {
    tx: mpsc::Sender<Command>,
    timeout: Duration,
    counter: AtomicU64,
    peer_id: String,
}

impl AppState {
    fn next_request_id(&self) -> String {
        let ts = SystemTime::now()
            .duration_since(UNIX_EPOCH)
            .unwrap_or(Duration::from_millis(0))
            .as_millis();
        let counter = self.counter.fetch_add(1, Ordering::Relaxed);
        format!("{}-{}-{}", self.peer_id, ts, counter)
    }
}

async fn handle_proxy_ask(
    axum::extract::State(state): axum::extract::State<Arc<AppState>>,
    Json(req): Json<ProxyAskRequest>,
) -> Result<Json<ProxyAskResponse>, (axum::http::StatusCode, String)> {
    let request_id = state.next_request_id();
    let ask = AskForProxy {
        request_id: request_id.clone(),
        deal_id: req.deal_id,
        provider: req.provider.trim().to_string(),
        file_path: req.file_path.trim().to_string(),
        range_start: req.range_start,
        range_len: req.range_len,
        max_price: req.max_price,
    };

    let (resp_tx, resp_rx) = oneshot::channel();
    state
        .tx
        .send(Command::RequestProxy {
            request: ask,
            respond_to: resp_tx,
        })
        .await
        .map_err(|_| (axum::http::StatusCode::SERVICE_UNAVAILABLE, "proxy queue closed".into()))?;

    let offer: ProxyOffer = match tokio::time::timeout(state.timeout, resp_rx).await {
        Ok(Ok(offer)) => offer,
        Ok(Err(_)) => {
            let _ = state
                .tx
                .send(Command::CancelProxy {
                    request_id: request_id.clone(),
                })
                .await;
            return Err((axum::http::StatusCode::BAD_GATEWAY, "proxy response dropped".into()));
        }
        Err(_) => {
            let _ = state
                .tx
                .send(Command::CancelProxy {
                    request_id: request_id.clone(),
                })
                .await;
            return Err((axum::http::StatusCode::GATEWAY_TIMEOUT, "proxy request timed out".into()));
        }
    };

    Ok(Json(ProxyAskResponse {
        request_id,
        deputy_peer_id: offer.deputy_peer_id,
        deputy_endpoint: offer.deputy_endpoint,
        price: offer.price,
    }))
}

#[tokio::main]
async fn main() -> Result<()> {
    tracing_subscriber::fmt()
        .with_env_filter(EnvFilter::from_default_env().add_directive("nil_p2p=info".parse().unwrap()))
        .init();

    let cli = Cli::parse();

    let (tx, rx) = mpsc::channel(32);
    let proxy_endpoint = cli.proxy_endpoint.trim().to_string();
    let node = NilNode::new(
        cli.seed,
        cli.port,
        rx,
        if proxy_endpoint.is_empty() { None } else { Some(proxy_endpoint) },
        cli.proxy_price,
    )
    .await?;
    
    // Spawn the node
    tokio::spawn(async move {
        if let Err(e) = node.run().await {
            eprintln!("Node error: {:?}", e);
        }
    });

    if !cli.http_addr.trim().is_empty() {
        let state = Arc::new(AppState {
            tx: tx.clone(),
            timeout: Duration::from_millis(cli.proxy_timeout_ms),
            counter: AtomicU64::new(1),
            peer_id: cli.seed.to_string(),
        });
        let app = Router::new()
            .route("/proxy/ask", post(handle_proxy_ask))
            .with_state(state);
        let addr = cli.http_addr.parse()?;
        let listener = tokio::net::TcpListener::bind(&addr).await?;
        tokio::spawn(async move {
            if let Err(err) = axum::serve(listener, app).await {
                eprintln!("proxy http server error: {:?}", err);
            }
        });
    }

    info!("Node started. Type 'announce <shard_id>' or 'dial <addr>'");

    // Simple REPL
    let stdin = std::io::stdin();
    let mut line = String::new();
    while stdin.read_line(&mut line).is_ok() {
        let parts: Vec<&str> = line.trim().split_whitespace().collect();
        if parts.is_empty() {
            continue;
        }

        match parts[0] {
            "announce" => {
                if parts.len() > 1 {
                    tx.send(Command::AnnounceShard { shard_id: parts[1].to_string() }).await?;
                } else {
                    println!("Usage: announce <shard_id>");
                }
            }
            "dial" => {
                if parts.len() > 1 {
                     if let Ok(addr) = parts[1].parse() {
                         tx.send(Command::Dial { addr }).await?;
                     } else {
                         println!("Invalid multiaddr");
                     }
                }
            }
            "proxy" => {
                if parts.len() > 5 {
                    let deal_id: u64 = parts[1].parse().unwrap_or(0);
                    let range_start: u64 = parts[4].parse().unwrap_or(0);
                    let range_len: u64 = parts[5].parse().unwrap_or(0);
                    let max_price: u64 = if parts.len() > 6 { parts[6].parse().unwrap_or(0) } else { 0 };
                    let (resp_tx, resp_rx) = oneshot::channel();
                    tx.send(Command::RequestProxy {
                        request: AskForProxy {
                            request_id: format!("repl-{}", deal_id),
                            deal_id,
                            provider: parts[2].to_string(),
                            file_path: parts[3].to_string(),
                            range_start,
                            range_len,
                            max_price,
                        },
                        respond_to: resp_tx,
                    }).await?;

                    match tokio::time::timeout(Duration::from_millis(cli.proxy_timeout_ms), resp_rx).await {
                        Ok(Ok(offer)) => println!("Proxy offer: {} {}", offer.deputy_peer_id, offer.deputy_endpoint),
                        Ok(Err(_)) => println!("Proxy response dropped"),
                        Err(_) => println!("Proxy request timed out"),
                    }
                } else {
                    println!("Usage: proxy <deal_id> <provider> <file_path> <range_start> <range_len> [max_price]");
                }
            }
            "quit" | "exit" => break,
            _ => println!("Unknown command"),
        }
        line.clear();
    }

    Ok(())
}
