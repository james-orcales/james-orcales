KCX Qwerty - Keyboard-Centric Experience 
========================================

QWERTY variant for keyboard-centric desktop navigation with programming and vim motions in mind.

- Streamlines keyboard shortcut usage with laptops, touch typists, 
vim users, and terminal hobbits in mind.
- Emulate vim motions ***everywhere***
- Additional layouts are provided to take advantage of keyboard layout switching. 

> [!NOTE]
> Check out my new layout, [Colejak][colejak]. It's better! 

Layouts
-------

### KCX (Qwerty)
![KCX (Qwerty) Layout Picture][kcx-qwerty-pic]

### KCX (Homerow Symbols)
![KCX (Homerow Symbols) Layout Picture][kcx-homerow-symbols-pic]

Quickstart
----------

### Installation

```sh
git clone https://github.com/jnz1g/kcx-qwerty
cd kcx-qwerty
mkdir --parents $HOME/.config/xkb/rules
mkdir --parents $HOME/.config/xkb/symbols
cp -r --suffix=.bak rules/* $HOME/.config/xkb/rules/
cp -r --suffix=.bak symbols/* $HOME/.config/xkb/symbols/
```

> [!NOTE]
> This will overwrite existing destination files. Your original files such as *evdev.xml*
will be saved as *evdev.xml.bak*

### Sway

```
set $kb "1:1:AT_Translated_Set_2_keyboard"
input $kb {
    xkb_layout kcx(qwerty),kcx(homerow_sym)
}
```

```
bindsym Control+semicolon exec swaymsg input $kb xkb_switch_layout next
```

My bash script to change focused window border colors when switching layouts.

```sh
#!/bin/sh

change_color() {
    case $1 in 
        "KCX (Qwerty)")
            swaymsg client.focused "#50FA7B #50FA7B #50FA7B #F8F8F2"
            ;;
        "KCX (Homerow Symbols)")
            swaymsg client.focused "#FF5555 #FF5555 #FF5555 #F8F8F2"
            ;;
    esac
}

swaymsg -t subscribe '["input"]' -m \
    | jq -r --unbuffered 'select(.change == "xkb_layout").input.xkb_active_layout_name' \
    | while read layout; do change_color "$layout"; done
```


### GNOME

1. Search for *KCX (Qwerty)* and *KCX (Homerow Symbols)* in `gnome-control-center > Keyboard > Input Sources`
2. Set the layout switch shortcut under `...Keyboard > View and Customize Shortcuts > Typing`
with my recommended *Control + semicolon* keybind.
3. Instead of using the full layout, you can toggle individual options in
`gnome-tweaks > Keyboard & Mouse > Additional Layout Options > KCX Options`

FAQ
---

### Why QWERTY instead of this other superior layout?

Qwerty is the standard. Learning an obscure layout for speed/ergonomics is unnecessary and inconvenient. 
On average, I type 110-135wpm which is more than enough (i flexed on ya btw). My style of touch typing
also maximizes ergonomics. *(I'll probably make a repo/wiki on this)*

### Where's Right Shift?

I never used it so it went bye-bye.

### Where's Right Alt? I need AltGr for foreign characters.

I do not use it so it was replaced with Escape. I suggest replacing Right Control with Right Alt instead
of Caps Lock in QWERTY mode if it is essential.

### Do I need a special keyboard for this?

Any standard keyboard will do. Flatter keyboards on laptops are better solely because it is easier
to press the relocated Escape with the right thumb. I find that bulky spacebars often get in the way.

### Copy Pasting in shell is buggy when using Homerow Symbols.

You might be doing `Control+RelocatedShift+{something}`. Instead, do 
`RelocatedShift+Control+{something}`. `Control+OriginalShift` and `OriginalShift+Control`
should both work properly still.
>
In KCX (Homerow Symbols), S is replaced with *[ Shift_L ]*. Despite this, 
`Control+RelocatedShift` becomes `Control+S` which sends a XOFF signal, freezing
the shell. `Control+q` will send XON and unfreeze the shell. There's probably a
way to configure this properly to prevent such behavior but eh...
>
In the meantime, XOFF/XON can be disabled  entirely by adding `stty -ixon` to 
`~/.bashrc` or `~/.zshrc`. To unfreeze the shell using any key, add `stty ixany` 
instead. [(source)][xoff/xon]

### Can I use a different base layout? (e.g. Dvorak, Colemak)

Yes. The remaps target key codes (keyboard location), not the represented 
symbols. The base layout can be changed inside `$HOME/.config/xkb/symbols/kcx`:
```
include "us(basic)" // replace this
include "us(dvorak)" // with this
```

> [!NOTE]
> layout, variant, and option labels are hardcoded so they won't
automatically reflect the actual symbols they are remapping. In Dvorak,
`kcxvar/homerow_sym(d_becomes_delete)` turns your `E` into `Delete`.

### Does this apply to TTY as well?
No. And I don't recommend doing anything this complicated to your TTY
settings... ***here be dragons***

Resources
---------

[XKB Configuration Files Documentation](https://www.charvolant.org/doug/xkb/html/node5.html#SECTION00054000000000000000)

[Libxkbcommon commit: Allow for custom rulesets through include files](https://github.com/xkbcommon/libxkbcommon/pull/108/commits/bc4a691cb9f45c3309c78c997e00212f0978d082)

[Setting up $HOME/.config/xkb/](https://who-t.blogspot.com/2020/02/user-specific-xkb-configuration-part-1.html)

[Creating evdev.xml](https://who-t.blogspot.com/2020/07/user-specific-xkb-configuration-part-2.html)

[Extending system layouts with custom variants](https://who-t.blogspot.com/2020/08/user-specific-xkb-configuration-part-3.html)

[Youtube video where I discovered the blog posts from](https://www.youtube.com/watch?v=utqpa_8SXkA)

[This only works on Wayland and XWayland](https://who-t.blogspot.com/2020/09/no-user-specific-xkb-configuration-in-x.html)

Acknowledgement
---------------

### Keyboard Layout Pictures
- Created with [keyboard-layout-editor.com][keyboard-layout-editor]. 
- JSON for all layouts are available in [assets][assets].

[kcx-qwerty-pic]: https://github.com/jnz1g/kcx-qwerty/blob/main/assets/kcx-qwerty.png
[kcx-homerow-symbols-pic]: https://github.com/jnz1g/kcx-qwerty/blob/main/assets/kcx-homerow-symbols.png
[assets]: https://github.com/jnz1g/kcx-qwerty/tree/main/assets
[colejak]: https://github.com/jnz1g/colejak

[keyboard-layout-editor]: http://www.keyboard-layout-editor.com/
[xoff/xon]: https://unix.stackexchange.com/a/12108/593070
