fn in_if() {
    if true {
        struct A {
            x: i32,
        }
        let _ = A { x: 0 };
    }
}

fn in_match() {
    match 0 {
        _ => {
            struct B {
                y: i32,
            }
            let _ = B { y: 0 };
        }
    }
}

fn in_closure() {
    let f = || {
        struct C {
            z: i32,
        }
        let _ = C { z: 0 };
    };
    let _ = f;
}
