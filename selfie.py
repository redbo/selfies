import RPi.GPIO as GPIO
import os
import threading
import time
import sys
import pygame
import pygame.font
import pygame.camera

os.environ["DISPLAY"] = ":0"
os.system("xset s off")
os.system("xset -dpms")
os.system("xset s noblank")

screensize = (900, 1600)
camsize = (320, 240)

pygame.font.init()
pygame.display.init()
screen = pygame.display.set_mode(screensize, pygame.FULLSCREEN | pygame.HWSURFACE | pygame.NOFRAME)
pygame.display.toggle_fullscreen()
pygame.mouse.set_visible(False)

flip = False
font = None
fontheight = 1000
font = pygame.font.Font('Raleway-Black.ttf', fontheight)

text1 = font.render("1", True, (255, 255, 255))
text2 = font.render("2", True, (255, 255, 255))
text3 = font.render("3", True, (255, 255, 255))
temp1 = pygame.Surface((text1.get_width(), text1.get_height())).convert()
temp2 = pygame.Surface((text2.get_width(), text2.get_height())).convert()
temp3 = pygame.Surface((text3.get_width(), text3.get_height())).convert()

clock = pygame.time.Clock()

camimage = None
cam = None
snap = False
button_pressed = None
# button_pressed = time.time() + 5

def button_callback(channel):
    global button_pressed
    time.sleep(0.01)
    if GPIO.input(10) == GPIO.HIGH:
        button_pressed = time.time()

GPIO.setwarnings(False)
GPIO.setmode(GPIO.BOARD)
GPIO.setup(10, GPIO.IN, pull_up_down=GPIO.PUD_DOWN)
GPIO.add_event_detect(10, GPIO.RISING, callback=button_callback)
GPIO.setup(3, GPIO.OUT)
GPIO.setup(5, GPIO.OUT)
GPIO.output(3, GPIO.HIGH)
GPIO.output(5, GPIO.HIGH)

def shiftsnaps():
    for _ in xrange(0, 25):
        snaps.scroll(18)
        time.sleep(0.1)

def focuson():
    GPIO.output(3, GPIO.LOW)
    time.sleep(5)

def camcam():
    global cam, snap
    pygame.camera.init()
    cam = pygame.camera.Camera("/dev/video0", camsize)
    hcam = pygame.camera.Camera("/dev/video0", (960, 720))
    cam.start()
    global camimage
    while True:
        if snap:
            time.sleep(0.1)
            GPIO.output(5, GPIO.LOW)
            time.sleep(0.1)
            GPIO.output(3, GPIO.HIGH)
            GPIO.output(5, GPIO.HIGH)
            cam.stop()
            hcam.start()
            img = hcam.get_image()
            if flip:
                img = pygame.transform.rotate(img, 180)
            snaps.blit(pygame.transform.scale(img, (450, 338)), (0, 0))
            threading.Thread(target=shiftsnaps).start()
            pygame.image.save(img, "snaps/" + str(time.time()) + ".jpg")
            hcam.stop()
            cam.start()
            snap = False
        else:
            camimage = cam.get_image()
            if flip:
                camimage = pygame.transform.rotate(camimage, 180)

threading.Thread(target=camcam).start()

while cam is None:
    time.sleep(1)

snaps = pygame.Surface((450 * 5, 338))
snaps.fill((0, 0, 0))
def blit_alpha(target, source, temp, opacity):
        location = source.get_rect()
        location.center = (screensize[0] / 2, screensize[1] / 2)
        temp.blit(target, (-location[0], -location[1]))
        temp.blit(source, (0, 0))
        temp.set_alpha(opacity)
        target.blit(temp, location)

while True:
    for event in pygame.event.get():
        if event.type == pygame.QUIT:
            sys.exit(0)
    pressed = pygame.key.get_pressed()
    if pressed[pygame.K_ESCAPE]:
        sys.exit(0)
    screen.fill((0, 0, 0))
    if camimage:
        screen.blit(pygame.transform.scale(camimage, (900, 675)), (0,0))
    screen.blit(snaps, (-450, 675 + 50))
    screen.blit(snaps, (-1350, 675 + 100 + 338))
    if button_pressed is not None:
        if time.time() - button_pressed > 4.75:
            screen.fill((255, 255, 255))
            button_pressed = None
        elif time.time() - button_pressed > 4.5:
            threading.Thread(target=focuson).start()
            screen.fill((255, 255, 255))
            snap = True
        elif time.time() - button_pressed > 3:
            blit_alpha(screen, text1, temp1, int(255 * ((4.5 - (time.time() - button_pressed)) / 1.5)))
        elif time.time() - button_pressed > 1.5:
            blit_alpha(screen, text2, temp2, int(255 * ((3 - (time.time() - button_pressed)) / 1.5)))
        elif time.time() - button_pressed > 0:
            blit_alpha(screen, text3, temp3, int(255 * ((1.5 - (time.time() - button_pressed)) / 1.5)))
    pygame.display.flip()
    clock.tick(10)

