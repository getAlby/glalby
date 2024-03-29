use std::str::FromStr;
use std::sync::Arc;
use std::time::Duration;
use tokio::time;

use anyhow::Context;
use bip39::Mnemonic;
use thiserror::Error;

use gl_client::bitcoin::Network;
use gl_client::pb::cln;
use gl_client::scheduler::Scheduler;
use gl_client::signer::model::greenlight::scheduler;
use gl_client::signer::Signer;
use gl_client::tls::TlsConfig;
use tokio::sync::mpsc::Sender;
use tokio::task::JoinHandle;

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
    pub gl_creds: String,
}

impl From<scheduler::RecoveryResponse> for GreenlightCredentials {
    fn from(recovery: scheduler::RecoveryResponse) -> Self {
        GreenlightCredentials {
            gl_creds: hex::encode(recovery.creds),
        }
    }
}

impl From<scheduler::RegistrationResponse> for GreenlightCredentials {
    fn from(registration: scheduler::RegistrationResponse) -> Self {
        GreenlightCredentials {
            gl_creds: hex::encode(registration.creds),
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
            color,
            network: info.network,
            block_height: info.blockheight,
            pubkey: hex::encode(info.id),
        }
    }
}

#[derive(Clone, Debug)]
pub struct ShutdownResponse {}

#[derive(Clone, Debug)]
pub struct MakeInvoiceRequest {
    pub amount_msat: u64,
    pub description: String,
    pub label: String,
    pub expiry: Option<u64>,
    pub fallbacks: Option<Vec<String>>,
    pub preimage: Option<String>,
    pub cltv: Option<u32>,
    pub deschashonly: Option<bool>,
}

impl TryFrom<MakeInvoiceRequest> for cln::InvoiceRequest {
    type Error = SdkError;

    fn try_from(req: MakeInvoiceRequest) -> Result<Self> {
        Ok(cln::InvoiceRequest {
            label: req.label,
            amount_msat: Some(cln::AmountOrAny {
                value: Some(cln::amount_or_any::Value::Amount(cln::Amount {
                    msat: req.amount_msat,
                })),
            }),
            description: req.description,
            expiry: req.expiry,
            fallbacks: req.fallbacks.unwrap_or(Vec::new()),
            preimage: req
                .preimage
                .map(hex::decode)
                .transpose()
                .context("preimage contains invalid hex value")
                .map_err(SdkError::invalid_arg)?,
            cltv: req.cltv,
            deschashonly: req.deschashonly,
        })
    }
}

#[derive(Clone, Debug)]
pub struct MakeInvoiceResponse {
    pub bolt11: String,
    pub payment_hash: String,
    pub payment_secret: String,
    pub expires_at: u64,
    pub created_index: Option<u64>,
    pub warning_capacity: Option<String>,
    pub warning_offline: Option<String>,
    pub warning_deadends: Option<String>,
    pub warning_private_unused: Option<String>,
    pub warning_mpp: Option<String>,
}

impl From<cln::InvoiceResponse> for MakeInvoiceResponse {
    fn from(invoice: cln::InvoiceResponse) -> Self {
        MakeInvoiceResponse {
            bolt11: invoice.bolt11,
            payment_hash: hex::encode(invoice.payment_hash),
            payment_secret: hex::encode(invoice.payment_secret),
            expires_at: invoice.expires_at,
            created_index: invoice.created_index,
            warning_capacity: invoice.warning_capacity,
            warning_offline: invoice.warning_offline,
            warning_deadends: invoice.warning_deadends,
            warning_private_unused: invoice.warning_private_unused,
            warning_mpp: invoice.warning_mpp,
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
pub struct TlvEntry {
    pub ty: u64,
    pub value: String,
}

impl TryFrom<TlvEntry> for cln::TlvEntry {
    type Error = SdkError;

    fn try_from(entry: TlvEntry) -> Result<Self> {
        Ok(cln::TlvEntry {
            r#type: entry.ty,
            value: hex::decode(entry.value)
                .context("TLV entry value contains invalid hex value")
                .map_err(SdkError::invalid_arg)?,
        })
    }
}

#[derive(Clone, Debug)]
pub struct KeySendRequest {
    pub destination: String,
    pub amount_msat: Option<u64>,
    pub label: Option<String>,
    pub extra_tlvs: Option<Vec<TlvEntry>>,
}

impl TryFrom<KeySendRequest> for cln::KeysendRequest {
    type Error = SdkError;

    fn try_from(req: KeySendRequest) -> Result<Self> {
        Ok(cln::KeysendRequest {
            destination: hex::decode(req.destination)
                .context("destination contains invalid hex value")
                .map_err(SdkError::invalid_arg)?,
            amount_msat: req.amount_msat.map(|a| cln::Amount { msat: a }),
            label: req.label,
            extratlvs: req
                .extra_tlvs
                .map(|tlvs| {
                    tlvs.into_iter()
                        .map(cln::TlvEntry::try_from)
                        .collect::<Result<_>>()
                })
                .transpose()?
                .map(|tlvs| cln::TlvStream { entries: tlvs }),
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
                .map_err(SdkError::invalid_arg)?,
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

#[derive(Copy, Clone, Debug)]
pub enum NewAddressType {
    Bech32,
    P2tr,
    All,
}

impl From<NewAddressType> for cln::newaddr_request::NewaddrAddresstype {
    fn from(t: NewAddressType) -> Self {
        match t {
            NewAddressType::Bech32 => cln::newaddr_request::NewaddrAddresstype::Bech32,
            NewAddressType::P2tr => cln::newaddr_request::NewaddrAddresstype::P2tr,
            NewAddressType::All => cln::newaddr_request::NewaddrAddresstype::All,
        }
    }
}

#[derive(Clone, Debug)]
pub struct NewAddressRequest {
    pub address_type: Option<NewAddressType>,
}

impl From<NewAddressRequest> for cln::NewaddrRequest {
    fn from(req: NewAddressRequest) -> Self {
        cln::NewaddrRequest {
            addresstype: req
                .address_type
                .map(cln::newaddr_request::NewaddrAddresstype::from)
                .map(|t| t as i32),
        }
    }
}

#[derive(Clone, Debug)]
pub struct NewAddressResponse {
    pub p2tr: Option<String>,
    pub bech32: Option<String>,
}

impl From<cln::NewaddrResponse> for NewAddressResponse {
    fn from(response: cln::NewaddrResponse) -> Self {
        NewAddressResponse {
            p2tr: response.p2tr,
            bech32: response.bech32,
        }
    }
}

#[derive(Copy, Clone, Debug)]
pub enum ListInvoicesIndex {
    Created,
    Updated,
}

impl From<ListInvoicesIndex> for cln::listinvoices_request::ListinvoicesIndex {
    fn from(i: ListInvoicesIndex) -> Self {
        match i {
            ListInvoicesIndex::Created => cln::listinvoices_request::ListinvoicesIndex::Created,
            ListInvoicesIndex::Updated => cln::listinvoices_request::ListinvoicesIndex::Updated,
        }
    }
}

#[derive(Clone, Debug)]
pub struct ListInvoicesRequest {
    pub label: Option<String>,
    pub invstring: Option<String>,
    pub payment_hash: Option<String>,
    pub offer_id: Option<String>,
    pub index: Option<ListInvoicesIndex>,
    pub start: Option<u64>,
    pub limit: Option<u32>,
}

impl TryFrom<ListInvoicesRequest> for cln::ListinvoicesRequest {
    type Error = SdkError;

    fn try_from(req: ListInvoicesRequest) -> Result<Self> {
        Ok(cln::ListinvoicesRequest {
            label: req.label,
            invstring: req.invstring,
            payment_hash: req
                .payment_hash
                .map(hex::decode)
                .transpose()
                .context("payment hash contains invalid hex value")
                .map_err(SdkError::invalid_arg)?,
            offer_id: req.offer_id,
            index: req
                .index
                .map(cln::listinvoices_request::ListinvoicesIndex::from)
                .map(|i| i as i32),
            start: req.start,
            limit: req.limit,
        })
    }
}

#[derive(Clone, Debug)]
pub struct ListInvoicesInvoicePaidOutpoint {
    pub txid: Option<String>,
    pub outnum: Option<u32>,
}

impl From<cln::ListinvoicesInvoicesPaidOutpoint> for ListInvoicesInvoicePaidOutpoint {
    fn from(outpoint: cln::ListinvoicesInvoicesPaidOutpoint) -> Self {
        ListInvoicesInvoicePaidOutpoint {
            txid: outpoint.txid.map(hex::encode),
            outnum: outpoint.outnum,
        }
    }
}

#[derive(Clone, Debug)]
pub struct ListInvoicesInvoice {
    pub label: String,
    pub description: Option<String>,
    pub payment_hash: String,
    pub status: i32,
    pub expires_at: u64,
    pub amount_msat: Option<u64>,
    pub bolt11: Option<String>,
    pub bolt12: Option<String>,
    pub local_offer_id: Option<String>,
    pub invreq_payer_note: Option<String>,
    pub created_index: Option<u64>,
    pub updated_index: Option<u64>,
    pub pay_index: Option<u64>,
    pub amount_received_msat: Option<u64>,
    pub paid_at: Option<u64>,
    pub paid_outpoint: Option<ListInvoicesInvoicePaidOutpoint>,
    pub payment_preimage: Option<String>,
}

impl From<cln::ListinvoicesInvoices> for ListInvoicesInvoice {
    fn from(invoice: cln::ListinvoicesInvoices) -> Self {
        ListInvoicesInvoice {
            label: invoice.label,
            description: invoice.description,
            payment_hash: hex::encode(invoice.payment_hash),
            status: invoice.status,
            expires_at: invoice.expires_at,
            amount_msat: invoice.amount_msat.map(|a| a.msat),
            bolt11: invoice.bolt11,
            bolt12: invoice.bolt12,
            local_offer_id: invoice.local_offer_id.map(hex::encode),
            invreq_payer_note: invoice.invreq_payer_note,
            created_index: invoice.created_index,
            updated_index: invoice.updated_index,
            pay_index: invoice.pay_index,
            amount_received_msat: invoice.amount_received_msat.map(|a| a.msat),
            paid_at: invoice.paid_at,
            paid_outpoint: invoice
                .paid_outpoint
                .map(ListInvoicesInvoicePaidOutpoint::from),
            payment_preimage: invoice.payment_preimage.map(hex::encode),
        }
    }
}

#[derive(Clone, Debug)]
pub struct ListInvoicesResponse {
    pub invoices: Vec<ListInvoicesInvoice>,
}

impl From<cln::ListinvoicesResponse> for ListInvoicesResponse {
    fn from(response: cln::ListinvoicesResponse) -> Self {
        ListInvoicesResponse {
            invoices: response
                .invoices
                .into_iter()
                .map(ListInvoicesInvoice::from)
                .collect(),
        }
    }
}

#[derive(Copy, Clone, Debug)]
pub enum ListPaymentsStatus {
    Pending,
    Complete,
    Failed,
}

impl From<ListPaymentsStatus> for cln::listpays_request::ListpaysStatus {
    fn from(s: ListPaymentsStatus) -> Self {
        match s {
            ListPaymentsStatus::Pending => cln::listpays_request::ListpaysStatus::Pending,
            ListPaymentsStatus::Complete => cln::listpays_request::ListpaysStatus::Complete,
            ListPaymentsStatus::Failed => cln::listpays_request::ListpaysStatus::Failed,
        }
    }
}

#[derive(Clone, Debug)]
pub struct ListPaymentsRequest {
    pub bolt11: Option<String>,
    pub payment_hash: Option<String>,
    pub status: Option<ListPaymentsStatus>,
}

impl TryFrom<ListPaymentsRequest> for cln::ListpaysRequest {
    type Error = SdkError;

    fn try_from(req: ListPaymentsRequest) -> Result<Self> {
        Ok(cln::ListpaysRequest {
            bolt11: req.bolt11,
            payment_hash: req
                .payment_hash
                .map(hex::decode)
                .transpose()
                .context("payment hash contains invalid hex value")
                .map_err(SdkError::invalid_arg)?,
            status: req
                .status
                .map(cln::listpays_request::ListpaysStatus::from)
                .map(|s| s as i32),
        })
    }
}

#[derive(Clone, Debug)]
pub struct ListPaymentsPayment {
    pub payment_hash: String,
    pub status: i32,
    pub destination: Option<String>,
    pub created_at: u64,
    pub completed_at: Option<u64>,
    pub label: Option<String>,
    pub bolt11: Option<String>,
    pub description: Option<String>,
    pub bolt12: Option<String>,
    pub amount_msat: Option<u64>,
    pub amount_sent_msat: Option<u64>,
    pub preimage: Option<String>,
    pub number_of_parts: Option<u64>,
    pub erroronion: Option<String>,
}

impl From<cln::ListpaysPays> for ListPaymentsPayment {
    fn from(payment: cln::ListpaysPays) -> Self {
        ListPaymentsPayment {
            payment_hash: hex::encode(payment.payment_hash),
            status: payment.status,
            destination: payment.destination.map(hex::encode),
            created_at: payment.created_at,
            completed_at: payment.completed_at,
            label: payment.label,
            bolt11: payment.bolt11,
            description: payment.description,
            bolt12: payment.bolt12,
            amount_msat: payment.amount_msat.map(|a| a.msat),
            amount_sent_msat: payment.amount_sent_msat.map(|a| a.msat),
            preimage: payment.preimage.map(hex::encode),
            number_of_parts: payment.number_of_parts,
            erroronion: payment.erroronion.map(hex::encode),
        }
    }
}

#[derive(Clone, Debug)]
pub struct ListPaymentsResponse {
    pub payments: Vec<ListPaymentsPayment>,
}

impl From<cln::ListpaysResponse> for ListPaymentsResponse {
    fn from(response: cln::ListpaysResponse) -> Self {
        ListPaymentsResponse {
            payments: response
                .pays
                .into_iter()
                .map(ListPaymentsPayment::from)
                .collect(),
        }
    }
}

#[derive(Clone, Debug)]
pub struct SignMessageRequest {
    pub message: String,
}

impl From<SignMessageRequest> for cln::SignmessageRequest {
    fn from(req: SignMessageRequest) -> Self {
        cln::SignmessageRequest {
            message: req.message,
        }
    }
}

#[derive(Clone, Debug)]
pub struct SignMessageResponse {
    pub signature: Vec<u8>,
    pub recid: Vec<u8>,
    pub zbase: String,
}

impl From<cln::SignmessageResponse> for SignMessageResponse {
    fn from(response: cln::SignmessageResponse) -> Self {
        SignMessageResponse {
            signature: response.signature,
            recid: response.recid,
            zbase: response.zbase,
        }
    }
}

#[derive(Copy, Clone, Debug)]
pub enum AmountOrAll {
    Amount { msat: u64 },
    All,
}

impl From<AmountOrAll> for cln::AmountOrAll {
    fn from(a: AmountOrAll) -> Self {
        match a {
            AmountOrAll::Amount { msat } => cln::AmountOrAll {
                value: Some(cln::amount_or_all::Value::Amount(cln::Amount { msat })),
            },
            AmountOrAll::All => cln::AmountOrAll {
                value: Some(cln::amount_or_all::Value::All(true)),
            },
        }
    }
}

#[derive(Clone, Debug)]
pub struct WithdrawRequest {
    pub destination: String,
    pub amount: Option<AmountOrAll>,
    pub minconf: Option<u32>,
}

impl From<WithdrawRequest> for cln::WithdrawRequest {
    fn from(req: WithdrawRequest) -> Self {
        cln::WithdrawRequest {
            destination: req.destination,
            satoshi: req.amount.map(AmountOrAll::into),
            minconf: req.minconf,
            ..Default::default()
        }
    }
}

#[derive(Clone, Debug)]
pub struct WithdrawResponse {
    pub tx: String,
    pub txid: String,
    pub psbt: String,
}

impl From<cln::WithdrawResponse> for WithdrawResponse {
    fn from(response: cln::WithdrawResponse) -> Self {
        WithdrawResponse {
            tx: hex::encode(response.tx),
            txid: hex::encode(response.txid),
            psbt: response.psbt,
        }
    }
}

#[derive(Clone, Debug)]
pub struct CloseRequest {
    pub id: String,
    pub unilateral_timeout: Option<u32>,
    pub destination: Option<String>,
    pub fee_negotiation_step: Option<String>,
    pub force_lease_closed: Option<bool>,
}

impl From<CloseRequest> for cln::CloseRequest {
    fn from(req: CloseRequest) -> Self {
        cln::CloseRequest {
            id: req.id,
            unilateraltimeout: req.unilateral_timeout,
            destination: req.destination,
            fee_negotiation_step: req.fee_negotiation_step,
            force_lease_closed: req.force_lease_closed,
            ..Default::default()
        }
    }
}

#[derive(Clone, Debug)]
pub struct CloseResponse {
    pub item_type: i32,
    pub tx: Option<String>,
    pub txid: Option<String>,
}

impl From<cln::CloseResponse> for CloseResponse {
    fn from(response: cln::CloseResponse) -> Self {
        CloseResponse {
            item_type: response.item_type,
            tx: response.tx.map(hex::encode),
            txid: response.txid.map(hex::encode),
        }
    }
}

pub struct GreenlightAlbyClient {
    node: gl_client::node::ClnClient,
    shutdown: Sender<()>,
    signer_handle: JoinHandle<()>,
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

pub async fn register(mnemonic: String, invite_code: String) -> Result<GreenlightCredentials> {
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
        .register(&signer, Some(invite_code))
        .await
        .context("failed to register node")
        .map_err(SdkError::greenlight_api)?
        .into())
}

pub async fn new_greenlight_alby_client(
    mnemonic: String,
    credentials: GreenlightCredentials,
) -> Result<Arc<GreenlightAlbyClient>> {
    let cred_bytes = hex::decode(&credentials.gl_creds)
        .context("failed to decode credentials")
        .map_err(SdkError::invalid_arg)?;

    let creds = gl_client::credentials::Builder::as_device()
        .from_bytes(&cred_bytes)
        .context("failed to parse credentials")
        .map_err(SdkError::invalid_arg)?
        .build()
        .context("failed to build credentials")
        .map_err(SdkError::greenlight_api)?;

    let tls = creds
        .tls_config()
        .context("failed to get TLS config from greenlight credentials")
        .map_err(SdkError::greenlight_api)?;

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

    let node = scheduler
        .node(creds.clone())
        .await
        .context("failed to create node")
        .map_err(SdkError::greenlight_api)
        .unwrap();

    let (tx, rx) = tokio::sync::mpsc::channel(1);
    let signer_handle = tokio::spawn(async move {
        println!("Run forever started");
        if let Err(e) = signer.run_forever(rx).await {
            eprintln!("Run forever error: {:?}", e);
        }
        println!("Run forever finished");
    });

    Ok(Arc::new(GreenlightAlbyClient {
        node,
        signer_handle,
        shutdown: tx,
    }))
}

impl GreenlightAlbyClient {
    pub async fn shutdown(&self) -> Result<ShutdownResponse> {
        println!("Sending shutdown message");
        self.shutdown.send(()).await.unwrap();

        let mut tries = 0;
        let max_tries = 5;
        while !self.signer_handle.is_finished() && tries < max_tries {
            println!("Waiting for signer to stop...");
            time::sleep(Duration::from_millis(1000)).await;
            tries += 1;
        }
        if tries == max_tries {
            println!("Shutdown failed, aborting handle");
            self.signer_handle.abort();
            time::sleep(Duration::from_millis(1000)).await;
        }

        println!("Greenlight shutdown finished");
        Ok(ShutdownResponse {})
    }

    pub async fn get_info(&self) -> Result<GetInfoResponse> {
        self.node
            .clone()
            .getinfo(cln::GetinfoRequest::default())
            .await
            .context("failed to get node info")
            .map_err(SdkError::greenlight_api)
            .map(|r| r.into_inner().into())
    }

    pub async fn make_invoice(&self, req: MakeInvoiceRequest) -> Result<MakeInvoiceResponse> {
        self.node
            .clone()
            .invoice(cln::InvoiceRequest::try_from(req)?)
            .await
            .context("failed to make invoice")
            .map_err(SdkError::greenlight_api)
            .map(|r| r.into_inner().into())
    }

    pub async fn pay(&self, req: PayRequest) -> Result<PayResponse> {
        self.node
            .clone()
            .pay(cln::PayRequest::from(req))
            .await
            .context("failed to pay invoice")
            .map_err(SdkError::greenlight_api)
            .map(|r| r.into_inner().into())
    }

    pub async fn key_send(&self, req: KeySendRequest) -> Result<KeySendResponse> {
        self.node
            .clone()
            .key_send(cln::KeysendRequest::try_from(req)?)
            .await
            .context("failed to send keysend")
            .map_err(SdkError::greenlight_api)
            .map(|r| r.into_inner().into())
    }

    pub async fn list_funds(&self, req: ListFundsRequest) -> Result<ListFundsResponse> {
        self.node
            .clone()
            .list_funds(cln::ListfundsRequest::from(req))
            .await
            .context("failed to list funds")
            .map_err(SdkError::greenlight_api)
            .map(|r| r.into_inner().into())
    }

    pub async fn connect_peer(&self, req: ConnectPeerRequest) -> Result<ConnectPeerResponse> {
        self.node
            .clone()
            .connect_peer(cln::ConnectRequest::from(req))
            .await
            .context("failed to connect peer")
            .map_err(SdkError::greenlight_api)
            .map(|r| r.into_inner().into())
    }

    pub async fn fund_channel(&self, req: FundChannelRequest) -> Result<FundChannelResponse> {
        self.node
            .clone()
            .fund_channel(cln::FundchannelRequest::try_from(req)?)
            .await
            .context("failed to fund channel")
            .map_err(SdkError::greenlight_api)
            .map(|r| r.into_inner().into())
    }

    pub async fn new_address(&self, req: NewAddressRequest) -> Result<NewAddressResponse> {
        self.node
            .clone()
            .new_addr(cln::NewaddrRequest::from(req))
            .await
            .context("failed to request new address")
            .map_err(SdkError::greenlight_api)
            .map(|r| r.into_inner().into())
    }

    pub async fn list_invoices(&self, req: ListInvoicesRequest) -> Result<ListInvoicesResponse> {
        self.node
            .clone()
            .list_invoices(cln::ListinvoicesRequest::try_from(req)?)
            .await
            .context("failed to list invoices")
            .map_err(SdkError::greenlight_api)
            .map(|r| r.into_inner().into())
    }

    pub async fn list_payments(&self, req: ListPaymentsRequest) -> Result<ListPaymentsResponse> {
        self.node
            .clone()
            .list_pays(cln::ListpaysRequest::try_from(req)?)
            .await
            .context("failed to list payments")
            .map_err(SdkError::greenlight_api)
            .map(|r| r.into_inner().into())
    }

    pub async fn sign_message(&self, req: SignMessageRequest) -> Result<SignMessageResponse> {
        self.node
            .clone()
            .sign_message(cln::SignmessageRequest::from(req))
            .await
            .context("failed to sign message")
            .map_err(SdkError::greenlight_api)
            .map(|r| r.into_inner().into())
    }

    pub async fn withdraw(&self, req: WithdrawRequest) -> Result<WithdrawResponse> {
        self.node
            .clone()
            .withdraw(cln::WithdrawRequest::from(req))
            .await
            .context("failed to withdraw")
            .map_err(SdkError::greenlight_api)
            .map(|r| r.into_inner().into())
    }

    pub async fn close(&self, req: CloseRequest) -> Result<CloseResponse> {
        self.node
            .clone()
            .close(cln::CloseRequest::from(req))
            .await
            .context("failed to close channel")
            .map_err(SdkError::greenlight_api)
            .map(|r| r.into_inner().into())
    }
}
