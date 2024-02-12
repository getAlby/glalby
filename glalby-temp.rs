use std::str::FromStr;

use gl_client::bitcoin::Network;
use gl_client::pb::cln;
use gl_client::scheduler::Scheduler;
use gl_client::signer::model::greenlight::scheduler;
use gl_client::signer::Signer;
use gl_client::tls::TlsConfig;

use bip39::Mnemonic;

#[tokio::main]
async fn main() {
    println!("hello world");
    let phrase = std::env::var("MNEMONIC").unwrap();
    let mnemonic = Mnemonic::from_str(&phrase).unwrap();
    // Prompt user to safely store the phrase

    let seed = &mnemonic.to_seed("")[0..32]; // Only need the first 32 bytes

    let secret = seed[0..32].to_vec();
    let tls = TlsConfig::new().unwrap();

    let signer = Signer::new(secret, Network::Bitcoin, tls).unwrap();

    let scheduler = Scheduler::new(signer.node_id(), Network::Bitcoin)
        .await
        .unwrap();

    let recover_res: scheduler::RecoveryResponse = scheduler.recover(&signer).await.unwrap();
    // TODO: store keys

    let tls = TlsConfig::new().unwrap().identity(
        recover_res.device_cert.into_bytes(),
        recover_res.device_key.into_bytes(),
    );
    let mut node: gl_client::node::ClnClient = scheduler.schedule(tls).await.unwrap();

    let info = node.getinfo(cln::GetinfoRequest::default()).await.unwrap();
    println!("{}", hex::encode(info.into_inner().id));
}
