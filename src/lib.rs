use std::sync::Arc;

use once_cell::sync::Lazy;

mod greenlight_alby_client;
use greenlight_alby_client::{
    new_greenlight_alby_client, GreenlightAlbyClient, GreenlightCredentials, Result, SdkError,
};

pub use greenlight_alby_client::{
    GetInfoResponse, ListFundsChannel, ListFundsOutput, ListFundsRequest, ListFundsResponse,
    MakeInvoiceRequest, MakeInvoiceResponse, PayRequest, PayResponse,
};

static RT: Lazy<tokio::runtime::Runtime> = Lazy::new(|| tokio::runtime::Runtime::new().unwrap());

pub struct BlockingGreenlightAlbyClient {
    greenlight_alby_client: Arc<GreenlightAlbyClient>,
}

impl BlockingGreenlightAlbyClient {
    pub fn get_info(&self) -> Result<GetInfoResponse> {
        rt().block_on(self.greenlight_alby_client.get_info())
    }

    pub fn make_invoice(&self, req: MakeInvoiceRequest) -> Result<MakeInvoiceResponse> {
        rt().block_on(self.greenlight_alby_client.make_invoice(req))
    }

    pub fn pay(&self, req: PayRequest) -> Result<PayResponse> {
        rt().block_on(self.greenlight_alby_client.pay(req))
    }

    pub fn list_funds(&self, req: ListFundsRequest) -> Result<ListFundsResponse> {
        rt().block_on(self.greenlight_alby_client.list_funds(req))
    }
}

pub fn recover(mnemonic: String) -> Result<GreenlightCredentials> {
    rt().block_on(greenlight_alby_client::recover(mnemonic))
}

pub fn new_blocking_greenlight_alby_client(
    mnemonic: String,
    credentials: GreenlightCredentials,
) -> Result<Arc<BlockingGreenlightAlbyClient>> {
    rt().block_on(async move {
        let greenlight_alby_client = new_greenlight_alby_client(mnemonic, credentials).await?;
        let blocking_greenlight_alby_client = Arc::new(BlockingGreenlightAlbyClient {
            greenlight_alby_client,
        });

        Ok(blocking_greenlight_alby_client)
    })
}

fn rt() -> &'static tokio::runtime::Runtime {
    &RT
}

uniffi::include_scaffolding!("glalby");
