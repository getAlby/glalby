use bip39::Mnemonic;
use gl_client::pb::cln::amount_or_any::Value;
use gl_client::pb::cln::{self, Amount, AmountOrAny};
use std::str::FromStr;
use std::sync::Arc;

use gl_client::bitcoin::Network;
use gl_client::scheduler::Scheduler;
use gl_client::signer::model::greenlight::scheduler;
use gl_client::signer::Signer;
use gl_client::tls::TlsConfig;

pub struct GreenlightCredentials {
    pub device_cert: String,
    pub device_key: String,
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

pub async fn recover(mnemonic: String) -> GreenlightCredentials {
    let mnemonic = Mnemonic::from_str(&mnemonic).unwrap();
    let seed = &mnemonic.to_seed("")[0..32]; // Only need the first 32 bytes

    let secret = seed[0..32].to_vec();
    let tls = TlsConfig::new().unwrap();

    let signer = Signer::new(secret, Network::Bitcoin, tls).unwrap();

    let scheduler = Scheduler::new(signer.node_id(), Network::Bitcoin)
        .await
        .unwrap();

    let recover_res: scheduler::RecoveryResponse = scheduler.recover(&signer).await.unwrap();
    return GreenlightCredentials {
        device_cert: recover_res.device_cert,
        device_key: recover_res.device_key,
    };
}

pub async fn new_greenlight_alby_client(
    mnemonic: String,
    credentials: GreenlightCredentials,
) -> Arc<GreenlightAlbyClient> {
    let tls = TlsConfig::new().unwrap().identity(
        credentials.device_cert.into_bytes(),
        credentials.device_key.into_bytes(),
    );

    let mnemonic = Mnemonic::from_str(&mnemonic).unwrap();
    let seed = &mnemonic.to_seed("")[0..32]; // Only need the first 32 bytes
    let secret = seed[0..32].to_vec();

    let signer = Signer::new(secret, Network::Bitcoin, tls.clone()).unwrap();
    let scheduler = Scheduler::new(signer.node_id(), Network::Bitcoin)
        .await
        .unwrap();

    Arc::new(GreenlightAlbyClient {
        tls,
        scheduler,
        // signer,
    })
}

impl GreenlightAlbyClient {
    async fn get_node(&self) -> gl_client::node::ClnClient {
        // wakes up the node
        let node: gl_client::node::ClnClient =
            self.scheduler.schedule(self.tls.clone()).await.unwrap();
        return node;
    }

    pub async fn get_info(&self) -> GreenlightNodeInfo {
        let mut node = self.get_node().await;

        // TODO: error handling, response handling
        let info = node
            .getinfo(cln::GetinfoRequest::default())
            .await
            .unwrap()
            .into_inner();

        info.into()
    }

    // TODO: change request, response type, add error handling
    pub async fn make_invoice(&self) -> String {
        let mut node = self.get_node().await;

        // TODO: error handling, response handling
        let invoice = node
            .invoice(cln::InvoiceRequest {
                label: rand::random::<u64>().to_string(),
                amount_msat: Some(AmountOrAny {
                    value: Some(Value::Amount(Amount { msat: 1000 })),
                }),
                ..Default::default()
            })
            .await
            .unwrap()
            .into_inner();

        println!("{}", invoice.bolt11);
        return invoice.bolt11;
    }
}
