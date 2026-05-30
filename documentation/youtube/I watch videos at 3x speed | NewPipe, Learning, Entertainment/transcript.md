# Watching YouTube at 3x Speed

There are two things you need to know about me.

I've been watching YouTube since I was a kid, in 2009. From the moment I wake up until bedtime, I'm
watching YouTube. Yes, I'm an iPad kid. And I'm just going to flex here a little: if you know the
game Bloons TD Battles, I was the number-one player in the country. The loadout was Cobra, Village,
and Boomerang, and then you rush the opponent on round 13 — it was unbeatable. I'm telling you,
YouTube is my primary source of knowledge and entertainment.

The second thing you need to know is that I like optimizing anything and everything, which led me to
watching videos at three times speed.

*This video is licensed under Creative Commons Attribution. All of my videos are unmonetized and
will remain so. I'm not sponsored by anyone, and I'm not affiliated with any entities mentioned in
this video.*

Our objective is to streamline accelerated media consumption, for the purposes of increasing our
entertainment throughput and our learning efficiency.

## YouTube clients

A YouTube client is some front-end application that lets you access YouTube. It provides a way to
watch videos and browse content with its own unique interface and feature set. The official YouTube
app is an example of a YouTube client; the website is also a YouTube client.

### The official YouTube app

Here, 2x playback speed comes with the free account. Recently they've introduced a feature — I think
in September 2025 — where YouTube Premium users can play up to 3x speed (sometimes I've even seen
4x). But that was only a recent addition; for most of the time that I've been watching 3x playback
speed, this wasn't a thing.

Number two: it resets the speed in various scenarios, probably because of bugs. For example, after
an advertisement plays, sometimes the playback speed gets reset. And when you exit the app or close
your browser, then when you reopen it, your playback speed is set to 1x again. Now, I don't know how
this works in YouTube Premium — whether it remembers your playback settings — but that's something
annoying with the official YouTube app.

### NewPipe

This is the YouTube client I use the most — like 99.99% of the time. This is what I use for watching
videos, because I only watch YouTube on my phone. It has 37,000 stars on GitHub, it's very popular,
and it allows 3x playback speed by default. It remembers your settings, but it's only for Android.
And if it matters to you, it's GPL-licensed.

This is how it looks. You have your feed; you can have subscriptions without creating an account,
which is nice; you can view the comments, download videos, play in the background, do
picture-in-picture mode — a lot of nice niceties in the app.

### FreeTube

Another YouTube client is FreeTube — 20,000 GitHub stars, very popular as well. It also allows 3x
playback speed. (Never mind the "Free" in the name — it's open source, and it's always going to be
free.) It also remembers playback speed. It's on Windows, macOS, and Linux, and it's AGPL-licensed.
I used this for a while when I was on Linux, but I eventually deleted it because I really don't
watch YouTube on desktop. This is how it looks — same idea: you can have subscriptions without
creating an account, and a bunch of other stuff.

### The browser extension

Now, if you don't want to download an actual client on your devices — maybe you don't trust them, or
you just don't like having a separate GUI for YouTube — there's another option: a browser extension.
I've seen it have a million users on Chrome; I don't remember the statistic on Firefox, but it
definitely has a lot of users. It allows more than 3x playback speed — actually, I don't think it
even has a limit. It remembers playback settings, and it's available in Chrome, Edge, and Firefox,
the default browsers on your major devices. (I didn't mention Safari here because I don't like
Safari and I don't have an idea of the browser-extension situation there. I did look at Dark Reader,
and you have to pay for that on Safari.)

Another thing: this browser extension is closed source — so if that's important to you... yeah,
because of this I don't use it anymore, but I did download it for this video. Lastly, the
distinguishing feature is that it works on embedded videos as well. Embedded videos are those
YouTube videos that appear in the content of some external website — for example, if you're reading
someone's personal blog and a YouTube video appears in the middle of their content. This extension
will apply your settings to that video, which is something you can't do with dedicated GUI clients
like NewPipe and FreeTube.

## Watching at 3x speed

### A demo

Let's actually start talking about watching 3x playback speed. I've prepared a video here by
ThePrimeagen. He's very popular among the software development community; he's somewhat of a fast
speaker, energetic. I've watched a significant portion of his channel this way — at least every
single video he's put up since 2023, I've watched. Okay, it's already set at 3x speed, and I'm going
to play it for you.

> Why am I choosing Go in 2024 over Rust? ...I've been choosing Go over and over and over again. One
> of the big things that's happened for me — doing Rust a lot for the last two years, for pretty
> much everything...

Okay. By the way, I understand this completely fine. The last time I watched this video was when it
was first uploaded two years ago, so it's not me just remembering what he was saying — it's not
fresh in my mind. But I completely understand what he's saying right now.

> ...And so I've been doing it for quite some time, and realizing that the type system is
> incredible. It is the best type system — it's like the second-best type system I've ever used. But
> there's a little something important to know about...

And you know what? I'm going to experiment — I'm going to try 4x for the first time here, and see if
I still understand. [plays at 4x] ...I actually understand him, which is crazy. I actually
understand him.

### How I built up to it

So, I know it sounds crazy. If this is your first time hearing this, it feels impossible — but I
didn't start out this way. I built up to it over four months. I started in October 2023, and I was
able to watch 3x speed and understand it by January 2024.

First I tried jumping up to 1.5x immediately, but it was too fast — I wasn't used to it. So I
started with 1.25x for two weeks. After two weeks I got up to 1.5x, and two weeks after that I
jumped straight to 2x. I just tried it — 2x is the big number, I didn't want to do 1.75x — and it
was fine; I didn't need a 1.75x incremental jump.

But for 2x speed, I got stuck here for a little while. I did the same thing — I tried jumping to
2.25x after two weeks, but it was too fast for me; my brain just wasn't used to the new speed. So I
stayed here for two months, and by January I tried playing 2.25x speed, and that felt the same as
2x. So I did the same thing I did with 1.5x: I jumped to 2.5x, did that for a week, and felt
confident that I understood things. So I jumped immediately to 3x after another week. And at this
point, yep, I understand 3x playback speed.

Now, it's easier for some people and harder for others, depending on how fast they speak, but
generally I could understand people at three times speed.

### The drawbacks

There's not much to say on the other playback speeds — I only saw drawbacks at 3x: fatigue, you
can't multitask, and a lot of people speak too fast.

**Fatigue.** There was one day I watched YouTube all day — I think eight to 10 hours straight, 3x
playback speed the whole time — and I was sleepy like five hours before my bedtime, which is crazy.
I'd never been sleepy that early in the day. So if you're going to watch 3x all the time, it's going
to take a toll on your energy and mental capacity eventually.

**You can't multitask.** That's because you need absolute attention on the video. The moment you
even look away or pay attention to your surroundings for a split second, you're going to miss like
three seconds of the video immediately, and you're just not going to comprehend anything coming into
your ears. I found this especially annoying when I was eating, because I like to watch YouTube while
eating — even just grabbing a bite is enough to break your focus, and you don't understand anything
that's happening. So when you're doing this, you really just have to be watching and doing nothing
else.

**A lot of people speak too fast.** I wouldn't say the majority, but there are enough that it's
annoying to use 3x as the default and then 2x for this person, 2.5x for that person. I'd say about 5
to 10% of people speak too fast for 3x playback speed, so it's annoying to keep that as the default.

### Why I default to 2x

So what I actually do is keep my default at 2x playback speed. I like it because, one, it's not
fatiguing — it's very easy for my brain. If I just showed you I could understand 4x, and I'm doing
half that speed, right? So this is second nature to me at this point. Two, I can multitask — I could
just have it in the background, doing something else, and still understand everything. And I'd say
99% of people speak slowly enough for 2x. There are like one to three people I need to put on 1.5x,
but most of the time they're completely fine.

## Entertainment throughput

Okay — what did this do for my entertainment throughput? Something simple, something relatable: in
August 2025, I watched all of these shows and all of their seasons. And you know what? I still had
time to go do stuff — I still had the rest of my day for my other stuff. And I still enjoyed the
shows, for three reasons.

One, the visuals help convey the context, reducing cognitive load compared to something like a book,
where you're really doing most of the work parsing the text, understanding what they mean, and the
underlying interpretation. With visuals, you just see, "Oh, they're in a desert," and that's it —
you don't need to watch that at 1x speed.

Two, I just want to know what happens — that's subjective for me. I'm not really the type to
interpret the scene. For example, in Breaking Bad, I don't look for the symbolism, so I miss a lot
of the artistic choices — like the stuff with Nacho. Wait, no — is it Nacho? Oh, that's Better Call
Saul. But yeah, you get the point.

## Learning efficiency

### YouTube as a learning platform

YouTube as a learning platform: the quality and quantity of videos depend solely on a topic's
popularity. For example, with Nix, there are a lot of good videos, but I'd say maybe 0.5% of
developers are even aware of Nix, and even fewer are using it. So, for example, web dev versus
firmware engineering — web dev is very popular.

### How YouTube shaped my learning path

My CS theory was ass — complete ass. I bombed all of my coding interviews, but I knew a lot of
technologies across domains and different parts of the software stack. So that meant I could create
a logging library in Go that does zero heap allocations, but I could not solve a scheduling
algorithm for an elevator in two hours. Yeah — that was one of my coding interviews: create a
scheduling algorithm for an elevator in two hours. And honestly, up until this point, I still don't
know how to do that.

I started in May 2023. I didn't know linked lists. I struggled with Java and the whole
object-oriented programming paradigm — you know, how primitive types can be whole objects with
methods on them, which confused the hell out of me. I don't know why people think object-oriented
programming is the best first language for beginners, but anyway. And then by December 2025, I
became the first hire at a startup in Berlin. My official title is software engineer, but my initial
focuses are on platform and infrastructure engineering.

So I'm 99% self-taught, primarily through YouTube. And this works for me for three reasons:
repetition is key, I know who to listen to, and I supplement myself with other forms of learning.

**Repetition is key.** That's just a general learning concept — you listen to things on repeat, you
keep doing the same stuff, you're technically practicing something, and that ingrains it into your
brain more. I'd compare it to writing: writing notes, compared to just typing them out on your
laptop, lets your brain absorb the material better because you're constantly looking at the text.

**Knowing who to listen to.** This is something you develop over time, and it definitely becomes
easier if you get good at a lot of different things. For example, I know a lot about music, I know a
lot about going to the gym — and I've learned how to look for the people who are focused on
educating and informing someone, instead of selling you something or just being hype.

**Supplement with other forms of learning.** Right now, as I'm getting more and more advanced, I'm
watching YouTube less and less, and most of the things that made me grow as a developer are actually
the books, the blogs, the forums, the projects, and following the developers I look up to. But this
would only be possible because I built a broad foundation through YouTube.

### Repetition in action

An example of "repetition is key" is this video by Alex Naskos on dynamic dispatch in a manually
memory-managed language — so, Zig. At the point when I watched this, I didn't even know about
lifetimes, so I was returning pointers and I was confused why I was getting a segmentation fault.
That's how bad I was at manual memory management. (Oh, by the way — rest in peace to Alex Naskos.)

Let's not watch it — I'm just going to show you how long this video is. I distinctly remember
watching this video at least six times. For the first four plays, I watched it the whole way
through, over and over, at 3x speed. And then for the parts I was confused by — especially this — I
remember I kept repeating this first part here. I put it back to normal speed and really tried to
understand what was happening. I even paused the video to read the code. So that's repetition is key
in action: you're basically skimming the whole video, absorbing everything the speaker has to say,
and then afterward you can evaluate which parts you want to go back to. Which is nice, because
honestly, for like 98% of the videos I watch, I'm completely fine watching them once — I don't need
to watch them a second time. Which definitely tells you something about the kind of information
available in these videos: it's very easy to digest.

### Beyond video

An example of other forms of learning: here is a podcast — so it's not only applicable to videos. I
know I listened to this podcast twice, at 2x speed. This is on Andrew Kelley and Ginger Bill. If you
don't know them: Andrew Kelley is the creator of the Zig programming language, and Ginger Bill is
the creator of the Odin programming language. Very polarizing philosophies on language design, so it
was a very interesting listen. I'll put the links in the description for everything I've shown here.

## The exceptions

So, when do I not watch things at 3x speed?

The obvious one would be, say, a music theory video where someone is always playing the piano in
between speaking. You're not going to watch that at 3x speed — you'd just mess up the intonation of
the music.

I also disable 3x speed when watching Vinh Giang and any kind of standup comedy. And you should try
guessing why. I'm going to pause for five seconds. Okay — it's because the delivery is more
important than the message itself. When you're watching Vinh Giang, yes, he's very good at
explaining what he does, but honestly, just looking at how he speaks — paying attention to how he
speaks — already gives you the lesson. So when he's talking about creating pauses in your speeches
to create emphasis on some point, you already get it. You don't need to listen to the words he's
actually saying — you just need to see how he's doing it. And the timing is important: the timing,
the rhythm, your intonation, your pitch — your pitch accent, rather.

And that's the same for standup comedy. It's a push and pull with the audience, and each comedian
has their own comedic timing. With 3x speed, you miss out on those moments — especially when they're
dropping a punchline — because everything is just one stream of words.

## Conclusion

So, the results.

**Streamlining accelerated media consumption:** use alternative YouTube clients to access faster
playback speeds, and set them as the default. 2x speed is the sweet spot.

**Learning efficiency:** it encourages repetition for deeper understanding, and it lowers the
barrier to entry for long-form content. I remember I saw this three-hour-long video on some software
development topic and thought, "Okay, I'll watch that — I could watch it in one hour at 3x playback
speed. I probably could watch it in 45 minutes, because there's definitely going to be some parts I
could skip over."

**Entertainment throughput:** the effectiveness depends on what the viewer wants out of the
experience. If the content is focused on the delivery of a message, it should be consumed normally.

What's out of scope for this video is knowing who or what to watch — like I said, that's something
you develop with experience, as you develop the variety of your knowledge and maintain a good
balance of learning materials.

In conclusion: accelerated media consumption increases entertainment throughput and learning
efficiency, and maximizing its effectiveness requires recognition of its use cases and weaknesses.

The end.
