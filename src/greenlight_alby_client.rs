use std::str::FromStr;
use std::sync::Arc;

use anyhow::Context;
use bip39::Mnemonic;
use thiserror::Error;

use gl_client::bitcoin::Network;
use gl_client::pb::cln::amount_or_any::Value;
use gl_client::pb::cln::{self, Amount, AmountOrAny};
use gl_client::scheduler::Scheduler;
use gl_client::signer::model::greenlight::scheduler;
use gl_client::signer::Signer;
use gl_client::tls::TlsConfig;

#[derive(Error, Debug)]
pub enum SdkError {
    #[error("invalid argument: {0}")]
    InvalidArgument(String),

    #[error("greenlight API error: {0}")]
    GreenlightApi(String),

    #[error("other error: {0}")]
    Other(String),
}

impl SdkError {
    fn invalid_arg(e: anyhow::Error) -> Self {
        SdkError::InvalidArgument(Self::format_anyhow_error(e))
    }

    fn greenlight_api(e: anyhow::Error) -> Self {
        SdkError::GreenlightApi(Self::format_anyhow_error(e))
    }

    fn other(e: anyhow::Error) -> Self {
        SdkError::Other(Self::format_anyhow_error(e))
    }

    fn format_anyhow_error(e: anyhow::Error) -> String {
        // Use alternate format (:#) to get the full error chain.
        format!("{:#}", e)
    }
}

pub type Result<T> = std::result::Result<T, SdkError>;

pub struct GreenlightCredentials {
    pub device_cert: String,
    pub device_key: String,
}

impl From<scheduler::RecoveryResponse> for GreenlightCredentials {
    fn from(recovery: scheduler::RecoveryResponse) -> Self {
        GreenlightCredentials {
            device_cert: recovery.device_cert,
            device_key: recovery.device_key,
        }
    }
}

#[derive(Clone, Debug)]
pub struct GreenlightNodeInfo {
    pub alias: String,
    pub color: String,
    pub network: String,
    pub block_height: u32,
}

impl From<cln::GetinfoResponse> for GreenlightNodeInfo {
    fn from(info: cln::GetinfoResponse) -> Self {
        GreenlightNodeInfo {
            alias: info.alias.unwrap_or_default(),
            color: hex::encode(info.color),
            network: info.network,
            block_height: info.blockheight,
        }
    }
}

pub struct GreenlightAlbyClient {
    // signer: gl_client::signer::Signer,
    scheduler: gl_client::scheduler::Scheduler,
    tls: TlsConfig,
}

pub async fn recover(mnemonic: String) -> Result<GreenlightCredentials> {
    let mnemonic = Mnemonic::from_str(&mnemonic)
        .context("failed to parse mnemonic")
        .map_err(SdkError::invalid_arg)?;

    let secret = mnemonic.to_seed("")[0..32].to_vec(); // Only need the first 32 bytes

    let tls = TlsConfig::new()
        .context("failed to create TLS config")
        .map_err(SdkError::greenlight_api)?;

    let signer = Signer::new(secret, Network::Bitcoin, tls)
        .context("failed to create signer")
        .map_err(SdkError::greenlight_api)?;

    let scheduler = Scheduler::new(signer.node_id(), Network::Bitcoin)
        .await
        .context("failed to create scheduler")
        .map_err(SdkError::greenlight_api)?;

    Ok(scheduler
        .recover(&signer)
        .await
        .context("failed to recover credentials")
        .map_err(SdkError::greenlight_api)?
        .into())
}

pub async fn new_greenlight_alby_client(
    mnemonic: String,
    credentials: GreenlightCredentials,
) -> Result<Arc<GreenlightAlbyClient>> {
    let tls = TlsConfig::new()
        .context("failed to create TLS config")
        .map_err(SdkError::greenlight_api)?
        .identity(
            credentials.device_cert.into_bytes(),
            credentials.device_key.into_bytes(),
        );

    let mnemonic = Mnemonic::from_str(&mnemonic)
        .context("failed to parse mnemonic")
        .map_err(SdkError::invalid_arg)?;

    let secret = mnemonic.to_seed("")[0..32].to_vec(); // Only need the first 32 bytes

    let signer = Signer::new(secret, Network::Bitcoin, tls.clone())
        .context("failed to create signer")
        .map_err(SdkError::greenlight_api)?;
    let scheduler = Scheduler::new(signer.node_id(), Network::Bitcoin)
        .await
        .context("failed to create scheduler")
        .map_err(SdkError::greenlight_api)?;

    Ok(Arc::new(GreenlightAlbyClient {
        tls,
        scheduler,
        // signer,
    }))
}

impl GreenlightAlbyClient {
    async fn get_node(&self) -> Result<gl_client::node::ClnClient> {
        // wakes up the node
        self.scheduler
            .schedule(self.tls.clone())
            .await
            .context("failed to schedule node")
            .map_err(SdkError::greenlight_api)
    }

    pub async fn get_info(&self) -> Result<GreenlightNodeInfo> {
        let mut node = self.get_node().await?;

        node.getinfo(cln::GetinfoRequest::default())
            .await
            .context("failed to get node info")
            .map_err(SdkError::greenlight_api)
            .map(|r| r.into_inner().into())
    }

    // TODO: change request, response type
    pub async fn make_invoice(&self) -> Result<String> {
        let mut node = self.get_node().await?;

        // TODO: response handling
        let invoice = node
            .invoice(cln::InvoiceRequest {
                label: rand::random::<u64>().to_string(),
                amount_msat: Some(AmountOrAny {
                    value: Some(Value::Amount(Amount { msat: 1000 })),
                }),
                ..Default::default()
            })
            .await
            .context("failed to make invoice")
            .map_err(SdkError::greenlight_api)?
            .into_inner();

        println!("{}", invoice.bolt11);
        Ok(invoice.bolt11)
    }
}
