#include <OneWire.h>
#include <DallasTemperature.h>

// Data wire is plugged into port 2 on the Arduino
#define ONE_WIRE_BUS 2

// Setup a oneWire instance to communicate with any OneWire devices (not just Maxim/Dallas temperature ICs)
OneWire oneWire(ONE_WIRE_BUS);

// Pass our oneWire reference to Dallas Temperature. 
DallasTemperature sensors(&oneWire);

DeviceAddress thermometer;
int deviceCount = 0;
void setup() {
  // put your setup code here, to run once:
  // start serial port
  Serial.begin(115200);
  Serial.println("Dallas Temperature Control Library - Async Demo");
  Serial.println("\nDemo shows the difference in length of the call\n\n");

  // Start up the library
  sensors.begin();  
  
  deviceCount = sensors.getDeviceCount();
  Serial.print("Found ");
  Serial.print(deviceCount, DEC);
  Serial.println("devices.");
  for (int i = 0; i < deviceCount; i++) { 
    if (!sensors.getAddress(thermometer, i)) { 
      Serial.print("Unable to find address for device");
      Serial.print(i, DEC);
      Serial.println(".");
    }
    Serial.print("Device at index ");
    Serial.print(i);
    Serial.print(" ");
    for(int j=0;j<sizeof(thermometer);j++) { 
      Serial.print(thermometer[j], HEX);
    }
    Serial.println(".");
  }

}

void loop() {
    sensors.requestTemperatures(); // Send the command to get temperatures

  float temp = sensors.getTempC(thermometer);
  Serial.print("Temperature");
  Serial.println(temp);
  // put your main code here, to run repeatedly:

}
