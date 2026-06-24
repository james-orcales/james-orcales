use std::fs;
use std::iter;
use std::net;
use std::sync::mpsc;

fn reads() {
    let _ = fs::read("input");
}

fn reads_text() {
    let _ = fs::read_to_string("input");
}

fn repeats() {
    let _ = iter::repeat(0u8);
}

fn connects() {
    let _ = net::TcpStream::connect("host");
}

fn receives(channel: mpsc::Receiver<u8>) {
    let _ = channel.recv();
}

fn accepts(listener: net::TcpListener) {
    let _ = listener.accept();
}
