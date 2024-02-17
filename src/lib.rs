use std::sync::Arc;

mod greenlight_alby_client;
use greenlight_alby_client::{
    new_greenlight_alby_client, GreenlightAlbyClient, GreenlightCredentials, GreenlightNodeInfo,
};

use once_cell::sync::Lazy;
static RT: Lazy<tokio::runtime::Runtime> = Lazy::new(|| tokio::runtime::Runtime::new().unwrap());

pub struct BlockingGreenlightAlbyClient {
    greenlight_alby_client: Arc<GreenlightAlbyClient>,
}

impl BlockingGreenlightAlbyClient {
    // TODO: change return type, add error handling
    pub fn get_info(&self) -> GreenlightNodeInfo {
        rt().block_on(self.greenlight_alby_client.get_info())
    }

    // TODO: change request type, return type, add error handling
    pub fn make_invoice(&self) -> String {
        rt().block_on(self.greenlight_alby_client.make_invoice())
    }
}

// TODO: error handling
pub fn recover(mnemonic: String) -> GreenlightCredentials {
    rt().block_on(greenlight_alby_client::recover(mnemonic))
}

// TODO: error handling
pub fn new_blocking_greenlight_alby_client(
    mnemonic: String,
    credentials: GreenlightCredentials,
) -> Arc<BlockingGreenlightAlbyClient> {
    rt().block_on(async move {
        let greenlight_alby_client = new_greenlight_alby_client(mnemonic, credentials).await;
        let blocking_greenlight_alby_client = Arc::new(BlockingGreenlightAlbyClient {
            greenlight_alby_client,
        });

        blocking_greenlight_alby_client
    })
}

fn rt() -> &'static tokio::runtime::Runtime {
    &RT
}

uniffi::include_scaffolding!("glalby");
