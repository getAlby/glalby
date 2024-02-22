use std::str::FromStr;
use std::sync::Arc;

use anyhow::Context;
use bip39::Mnemonic;
use thiserror::Error;

use gl_client::bitcoin::Network;
use gl_client::pb::cln;
use gl_client::scheduler::Scheduler;
use gl_client::signer::model::greenlight::scheduler;
use gl_client::signer::Signer;
use gl_client::tls::TlsConfig;

#[derive(Error, Clone, Debug)]
pub enum SdkError {
    #[error("invalid argument: {0}")]
    InvalidArgument(String),

    #[error("greenlight API error: {0}")]
    GreenlightApi(String),
    // #[error("other error: {0}")]
    // Other(String),
}

impl SdkError {
    fn invalid_arg(e: anyhow::Error) -> Self {
        SdkError::InvalidArgument(Self::format_anyhow_error(e))
    }

    fn greenlight_api(e: anyhow::Error) -> Self {
        SdkError::GreenlightApi(Self::format_anyhow_error(e))
    }

    // fn other(e: anyhow::Error) -> Self {
    //     SdkError::Other(Self::format_anyhow_error(e))
    // }

    fn format_anyhow_error(e: anyhow::Error) -> String {
        // Use alternate format (:#) to get the full error chain.
        format!("{:#}", e)
    }
}

pub type Result<T> = std::result::Result<T, SdkError>;

#[derive(Clone, Debug)]
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
pub struct GetInfoResponse {
    pub pubkey: String,
    pub alias: String,
    pub color: String,
    pub network: String,
    pub block_height: u32,
}

impl From<cln::GetinfoResponse> for GetInfoResponse {
    fn from(info: cln::GetinfoResponse) -> Self {
        let mut color = String::from("#");
        color.push_str(&hex::encode(info.color));
        GetInfoResponse {
            alias: info.alias.unwrap_or_default(),
            color: color,
            network: info.network,
            block_height: info.blockheight,
            pubkey: hex::encode(info.id),
        }
    }
}

#[derive(Clone, Debug)]
pub struct MakeInvoiceRequest {
    pub amount_msat: u64,
    pub description: String,
    pub label: String,
}

impl From<MakeInvoiceRequest> for cln::InvoiceRequest {
    fn from(req: MakeInvoiceRequest) -> Self {
        cln::InvoiceRequest {
            label: req.label,
            amount_msat: Some(cln::AmountOrAny {
                value: Some(cln::amount_or_any::Value::Amount(cln::Amount {
                    msat: req.amount_msat,
                })),
            }),
            description: req.description,
            ..Default::default()
        }
    }
}

#[derive(Clone, Debug)]
pub struct MakeInvoiceResponse {
    pub bolt11: String,
}

impl From<cln::InvoiceResponse> for MakeInvoiceResponse {
    fn from(invoice: cln::InvoiceResponse) -> Self {
        MakeInvoiceResponse {
            bolt11: invoice.bolt11,
        }
    }
}

#[derive(Clone, Debug)]
pub struct PayRequest {
    pub bolt11: String,
}

impl From<PayRequest> for cln::PayRequest {
    fn from(req: PayRequest) -> Self {
        cln::PayRequest {
            bolt11: req.bolt11,
            ..Default::default()
        }
    }
}

#[derive(Clone, Debug)]
pub struct PayResponse {
    pub preimage: String,
}

impl From<cln::PayResponse> for PayResponse {
    fn from(pay: cln::PayResponse) -> Self {
        PayResponse {
            preimage: hex::encode(pay.payment_preimage),
        }
    }
}

#[derive(Clone, Debug)]
pub struct KeySendRequest {
    pub destination: String,
    pub amount_msat: Option<u64>,
    pub label: Option<String>,
}

impl TryFrom<KeySendRequest> for cln::KeysendRequest {
    type Error = SdkError;

    fn try_from(req: KeySendRequest) -> Result<Self> {
        Ok(cln::KeysendRequest {
            destination: hex::decode(req.destination)
                .context("destination contains invalid hex value")
                .map_err(SdkError::invalid_arg)?
                .into(),
            amount_msat: req.amount_msat.map(|a| cln::Amount { msat: a }),
            label: req.label,
            ..Default::default()
        })
    }
}

#[derive(Clone, Debug)]
pub struct KeySendResponse {
    pub payment_preimage: String,
}

impl From<cln::KeysendResponse> for KeySendResponse {
    fn from(pay: cln::KeysendResponse) -> Self {
        KeySendResponse {
            payment_preimage: hex::encode(pay.payment_preimage),
        }
    }
}

#[derive(Clone, Debug)]
pub struct ListFundsRequest {
    pub spent: Option<bool>,
}

impl From<ListFundsRequest> for cln::ListfundsRequest {
    fn from(req: ListFundsRequest) -> Self {
        cln::ListfundsRequest { spent: req.spent }
    }
}

#[derive(Clone, Debug)]
pub struct ListFundsOutput {
    pub txid: String,
    pub output: u32,
    pub amount_msat: Option<u64>,
    pub scriptpubkey: String,
    pub address: Option<String>,
    pub redeemscript: Option<String>,
    pub status: i32,
    pub reserved: bool,
    pub blockheight: Option<u32>,
}

impl From<cln::ListfundsOutputs> for ListFundsOutput {
    fn from(output: cln::ListfundsOutputs) -> Self {
        ListFundsOutput {
            txid: hex::encode(output.txid),
            output: output.output,
            amount_msat: output.amount_msat.map(|a| a.msat),
            scriptpubkey: hex::encode(output.scriptpubkey),
            address: output.address,
            redeemscript: output.redeemscript.map(hex::encode),
            status: output.status,
            reserved: output.reserved,
            blockheight: output.blockheight,
        }
    }
}

#[derive(Clone, Debug)]
pub struct ListFundsChannel {
    pub peer_id: String,
    pub our_amount_msat: Option<u64>,
    pub amount_msat: Option<u64>,
    pub funding_txid: String,
    pub funding_output: u32,
    pub connected: bool,
    pub state: i32,
    pub channel_id: Option<String>,
    pub short_channel_id: Option<String>,
}

impl From<cln::ListfundsChannels> for ListFundsChannel {
    fn from(channel: cln::ListfundsChannels) -> Self {
        ListFundsChannel {
            peer_id: hex::encode(channel.peer_id),
            our_amount_msat: channel.our_amount_msat.map(|a| a.msat),
            amount_msat: channel.amount_msat.map(|a| a.msat),
            funding_txid: hex::encode(channel.funding_txid),
            funding_output: channel.funding_output,
            connected: channel.connected,
            state: channel.state,
            channel_id: channel.channel_id.map(hex::encode),
            short_channel_id: channel.short_channel_id,
        }
    }
}

#[derive(Clone, Debug)]
pub struct ListFundsResponse {
    pub outputs: Vec<ListFundsOutput>,
    pub channels: Vec<ListFundsChannel>,
}

impl From<cln::ListfundsResponse> for ListFundsResponse {
    fn from(response: cln::ListfundsResponse) -> Self {
        ListFundsResponse {
            outputs: response
                .outputs
                .into_iter()
                .map(ListFundsOutput::from)
                .collect(),
            channels: response
                .channels
                .into_iter()
                .map(ListFundsChannel::from)
                .collect(),
        }
    }
}

#[derive(Clone, Debug)]
pub struct ConnectPeerRequest {
    pub id: String,
    pub host: Option<String>,
    pub port: Option<u16>,
}

impl From<ConnectPeerRequest> for cln::ConnectRequest {
    fn from(req: ConnectPeerRequest) -> Self {
        cln::ConnectRequest {
            id: req.id,
            host: req.host,
            port: req.port.map(|p| p as u32),
        }
    }
}

#[derive(Clone, Debug)]
pub struct ConnectPeerResponse {
    pub id: String,
}

impl From<cln::ConnectResponse> for ConnectPeerResponse {
    fn from(response: cln::ConnectResponse) -> Self {
        ConnectPeerResponse {
            id: hex::encode(response.id),
        }
    }
}

#[derive(Clone, Debug)]
pub struct FundChannelRequest {
    pub id: String,
    pub amount_msat: Option<u64>,
    pub announce: Option<bool>,
    pub minconf: Option<u32>,
}

impl TryFrom<FundChannelRequest> for cln::FundchannelRequest {
    type Error = SdkError;

    fn try_from(req: FundChannelRequest) -> Result<Self> {
        Ok(cln::FundchannelRequest {
            id: hex::decode(req.id)
                .context("channel id contains invalid hex value")
                .map_err(SdkError::invalid_arg)?
                .into(),
            amount: req.amount_msat.map(|a| cln::AmountOrAll {
                value: Some(cln::amount_or_all::Value::Amount(cln::Amount { msat: a })),
            }),
            announce: req.announce,
            minconf: req.minconf,
            ..Default::default()
        })
    }
}

#[derive(Clone, Debug)]
pub struct FundChannelResponse {
    pub txid: String,
}

impl From<cln::FundchannelResponse> for FundChannelResponse {
    fn from(response: cln::FundchannelResponse) -> Self {
        FundChannelResponse {
            txid: hex::encode(response.txid),
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

    pub async fn get_info(&self) -> Result<GetInfoResponse> {
        let mut node = self.get_node().await?;

        node.getinfo(cln::GetinfoRequest::default())
            .await
            .context("failed to get node info")
            .map_err(SdkError::greenlight_api)
            .map(|r| r.into_inner().into())
    }

    pub async fn make_invoice(&self, req: MakeInvoiceRequest) -> Result<MakeInvoiceResponse> {
        let mut node = self.get_node().await?;

        node.invoice(cln::InvoiceRequest::from(req))
            .await
            .context("failed to make invoice")
            .map_err(SdkError::greenlight_api)
            .map(|r| r.into_inner().into())
    }

    pub async fn pay(&self, req: PayRequest) -> Result<PayResponse> {
        let mut node = self.get_node().await?;

        node.pay(cln::PayRequest::from(req))
            .await
            .context("failed to pay invoice")
            .map_err(SdkError::greenlight_api)
            .map(|r| r.into_inner().into())
    }

    pub async fn key_send(&self, req: KeySendRequest) -> Result<KeySendResponse> {
        let mut node = self.get_node().await?;

        node.key_send(cln::KeysendRequest::try_from(req)?)
            .await
            .context("failed to send keysend")
            .map_err(SdkError::greenlight_api)
            .map(|r| r.into_inner().into())
    }

    pub async fn list_funds(&self, req: ListFundsRequest) -> Result<ListFundsResponse> {
        let mut node = self.get_node().await?;

        node.list_funds(cln::ListfundsRequest::from(req))
            .await
            .context("failed to list funds")
            .map_err(SdkError::greenlight_api)
            .map(|r| r.into_inner().into())
    }

    pub async fn connect_peer(&self, req: ConnectPeerRequest) -> Result<ConnectPeerResponse> {
        let mut node = self.get_node().await?;

        node.connect_peer(cln::ConnectRequest::from(req))
            .await
            .context("failed to connect peer")
            .map_err(SdkError::greenlight_api)
            .map(|r| r.into_inner().into())
    }

    pub async fn fund_channel(&self, req: FundChannelRequest) -> Result<FundChannelResponse> {
        let mut node = self.get_node().await?;

        node.fund_channel(cln::FundchannelRequest::try_from(req)?)
            .await
            .context("failed to fund channel")
            .map_err(SdkError::greenlight_api)
            .map(|r| r.into_inner().into())
    }
}
