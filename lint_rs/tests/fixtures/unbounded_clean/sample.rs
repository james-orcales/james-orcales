use std::iter;
use std::net;
use std::sync::mpsc;
use std::time;

fn repeats() {
    let _ = iter::repeat_n(0u8, 4);
}

fn connects(address: net::SocketAddr, timeout: time::Duration) {
    let _ = net::TcpStream::connect_timeout(&address, timeout);
}

fn receives(channel: mpsc::Receiver<u8>) {
    let _ = channel.try_recv();
}
