# Selfies

This is a raspberry pi photobooth application I wrote for my wedding.  The wedding is now over, but I'm still playing around with the code.  It is not very finished and is currently fairly specific to my hardware, so it's probably not what you want.

After writing many variants, the only one that I could get to perform well enough was based on go-sdl2.  Implementations based on ebiten and pixel were rendering a webcam view at less than 5 fps while the sdl solution can hit upwards of 60fps, faster than the webcam can capture images anyway.

My hardware list:

  - A raspberry pi.
  - A relay board.
  - I have two large, arcade-style buttons mounted in a wooden box.
  - An arduino to interface with the relays and buttons.  The raspberry pi's GPIO would work, but I have a pile of dead pis from doing things like accidentally shorting pins.
  - A light weight 900x1600 monitor (that I got for about $25 at goodwill).
  - A logitech c922 variant webcam.  I also had a small form factor mirrorless camera that I triggered with a shutter release adapter plugged into the relay, but it failed to capture very many images, for various reasons.
  - A flash.  I found a bright flashlight on sale, and soldered relay leads to the on button. I covered the business end with packing material to diffuse the light.
