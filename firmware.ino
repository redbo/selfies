#define LEN(x) (sizeof(x)/sizeof((x)[0]))

unsigned long buttonDown[7] = {0, 0, 0, 0, 0, 0, 0};
int buttonPressed[7] = {0, 0, 0, 0, 0, 0, 0};
const int buttonPin[7] = {2, 3, 4, 5, 6, 7, 8};
const int relayPin[4] = {9, 10, 11, 12};
const int ledPin = 13;

void setup() {
  for (int i = 0; i < LEN(buttonPin); i++) {
    pinMode(buttonPin[i], INPUT);
    digitalWrite(buttonPin[i], HIGH); // turn on pullup resistor
  }
  pinMode(ledPin, OUTPUT);
  Serial.begin(9600);
}

void loop() {
  if (Serial.available() > 0) {
    switch (int c = Serial.read()) {
    case 'A': case 'B': case 'C': case 'D':
      digitalWrite(relayPin[c - 'A'], HIGH);
      break;
    case 'a': case 'b': case 'c': case 'd':
      digitalWrite(relayPin[c - 'A'], LOW);
      break;
    case 'R':
      for (int i = 0; i < LEN(relayPin); i++) {
        digitalWrite(relayPin[i], LOW);
      }
      break;
    }
  }

  for (int i = 0; i < LEN(buttonPin); i++) {
    if (digitalRead(buttonPin[i]) == 1 && buttonDown[i] == 0) {
      buttonDown[i] = millis();
    } else if (digitalRead(buttonPin[i]) == 0 && buttonDown[i] != 0) {
      buttonDown[i] = 0;
      buttonPressed[i] = 0;
    }

    if (buttonDown[i] > 0 && !buttonPressed[i] && (millis() - buttonDown[i]) > 25) {
      Serial.println(buttonPin[i]);
      buttonPressed[i] = 1;
    }
  }

  int anyPressed = 0;
  for (int i = 0; i < LEN(buttonPressed); i++) {
    anyPressed = anyPressed && buttonPressed[i];
  }
  digitalWrite(ledPin, anyPressed ? HIGH : LOW);

  delay(5);
}

