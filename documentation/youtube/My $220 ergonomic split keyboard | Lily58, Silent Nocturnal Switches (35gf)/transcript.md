# The Lily58: A Custom Split Mechanical Keyboard

## What it is

This is the Lily58. It's a custom mechanical split keyboard. I got one with Choc switches, which
makes it very thin — around 1.5 inches thick if you include the rubber feet at the bottom.

This is the QMK version, so it's always wired. This is the aux cable that connects the two sides,
and then you connect it to your computer with this USB-C to USB-C cable. I got these from Posture
Atelier (links in the description — I'm not sponsored by them). This is an open-source design;
anyone can make this keyboard.

I got these for $230 (13,000 Philippine pesos), pre-soldered, so I still needed to do a little bit
of assembly on my own. You can get it as low as $100 if you're willing to do all of the soldering.
They come with these Silent Nocturnal switches, 35 grams of force, but you can choose other
switches.

I've had these for 3 months and I rate them a 9.5 out of 10. I only have two problems with them:
one, there are too many keys; and two, the thumb cluster.

## Two things I'd change

### 1. Too many keys

Let's start with number one: too many keys. This is a 58-key keyboard, and I'm specifically talking
about these two keys right here — I think they're unnecessary. (This one's a bit debatable, so we'll
focus on these two right here. They're also present on the other side, these ones.)

These buttons are way too out of reach. Whatever functionality you're thinking of putting on them,
it's not going to be something you'll need on a regular basis — and if it is, then I'd wager it's
better to put that on another layer. Maybe you have a use case for it. I tried to find one; I
couldn't. Not that big of a deal, though.

### 2. The thumb key

Let's move on to the next problem: the thumb key. I would prefer something more like the ZSA
Voyager. On the ZSA Voyager, there's another thumb key out there jutting out from the keyboard.

Here's my setup: this is my spacebar and this is my shift key, and I rest my thumb on my spacebar.
Now, if I rest my hand on the keyboard, it looks like this. If you'll notice, it's angled out a bit,
which defeats the purpose of an ortholinear keyboard.

Now, you might say, "Well, James, just rest your hand or your thumb on the outer key." And I'll tell
you, it doesn't really matter where you rest your thumb here, because assuming you use both of these
keys at a high enough frequency, you will eventually angle out your hand so you can switch between
the two buttons faster.

Besides that, I have nothing else to say. It's a great keyboard. It's light, it's portable, it's as
big as my thumb, and it just works — which is the best thing you can say about anything, really.

## The switches

Let's talk about the switches. The Silent Nocturnal switches are light, quiet, and amazing to type
on. These are my very first linear switches. There's a 20g version, and I wouldn't recommend that —
oftentimes when I'm just resting my hand on the keyboard, I'll be pressing down the buttons without
even realizing it. With a 20-gram version that would be more frequent, and honestly it would be
annoying.

It's quiet — not dead silent, but quiet enough. For comparison, I have my old laptop here. It's an
Asus VivoBook or Zenbook, I don't know; the model number is X540UP. It's quieter than this laptop —
a $600 laptop from 2016, pretty loud. I'm recording from a MacBook Air, an M4 MacBook Air.

Now, the Lily58 compared to the M4 MacBook Air keyboard: when it comes to loudness, I'd say they're
pretty much equal. The difference is the tone. The MacBook Air keyboard is brighter, and this has a
deeper tone to it. In a dead silent environment like this, you will hear it, but with a little
background noise it's pretty much negligible.

With these switches, you really are your own worst enemy when it comes to the sound. If you don't
want to make any noise, it's possible — just control your presses and it'll be super quiet. But if
you're like me, I press these keys super hard. They always hit the PCB, which is pretty noisy. I
type this loud, yeah.

## QMK firmware

Let's talk about the QMK firmware now, which is the thing that gets me most excited about custom
keyboards. I actually run a custom keyboard layout here — I'll show you. It's a variant, a modified
version of Colemak-DH, optimized for programming. With QMK, I can remap the layout of this keyboard
at the firmware level. So whenever I connect this keyboard to any device, it's going to have my
custom layout, which is amazing.

Beforehand, I would have needed to do software remaps. On Linux that would be XKB configuration, and
on macOS it would be Karabiner-Elements. If you want a minimalist setup, maybe software remaps are
the thing for you. But with the added benefits of the hardware itself — the ortholinear layout and
split design — it's worth it to me.

Split keyboards aren't the only thing with QMK firmware. If you know the brand Keychron, they also
make just your standard ANSI keyboards — the long ones with staggered keys, similar to a laptop
keyboard, but with QMK firmware. So if you don't want to dive too deep into the rabbit hole but you
still want to do a little bit of keyboard remapping, there you go.

## Flashing the keyboard

When it comes to flashing the keyboard, it's pretty straightforward. I'm not going to do a full
tutorial here, I'm just going to tell you the general process.

Take your USB-C, connect it to your keyboard, connect it to your computer. Once your keyboard is
turned on, you see this button right here — press it and hold it down for a couple of seconds, and
that will enter bootloader mode. Now on your computer (File Explorer, Finder), another device will
show up. It's a USB storage device — that's your keyboard in bootloader mode. Open that, and then —
not enter — copy your compiled firmware onto the device, and as soon as you paste it, bam, your
firmware is flashed.

And what compiled firmware am I talking about? I'll show you my screen again. Here in my repository
— this is my repository, `james-regardless-jack` — scroll down, and here in the QMK section I just
have a script that sets everything up for you. It downloads the QMK CLI. You also need to clone my
repository for this, because that's where my configuration is: `keymap.c`, this is my configuration.

So you just execute the script, and it will open up your file explorer in the directory where the
compiled firmware is. Then you just copy that firmware onto your keyboard, and there you go — your
keyboard is flashed. Then you repeat the process for the other side of the keyboard, and that's it.
You don't have to worry about the firmware on this keyboard ever again.

## Verdict

To bring it back: the Lily58 is a 9.5 out of 10. I have a couple of problems with it, but if I could
live with this keyboard forever, I'd be fine with it. Does it warrant buying the ZSA Voyager, which
is at least another $365? No. Will I still buy the ZSA Voyager? Probably.

That's it. Thank you for watching.
