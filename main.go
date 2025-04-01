package main

import (
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/tarm/serial"
)

// эмуляция АЦП
func getADCValue() uint16 {
	val := uint16(rand.Intn(1023))
	return val
}

// Функция, принимающая 10-битное число и возвращающая два байта: low6 и high4
func getLow6AndHigh4(value uint16) (byte, byte) {
	// Убедимся, что значение действительно 10-битное
	if value > 1023 {
		panic("Value must be 10 bits or less (0-1023)")
	}

	// Для low6 первые два бита равны 1, остальные 6 бит из value
	low6 := byte(0xC0 | (value & 0x3F)) // 0xC0 = 11000000, сохраняем старшие два бита как 1

	// Для high4 используем старшие 4 бита
	high4 := byte((value >> 6) & 0x0F) // Сдвигаем на 6 бит и берем старшие 4 бита

	return low6, high4
}

func main() {
	// Конфигурация COM-порта
	config := &serial.Config{
		Name:     "COM12", // Укажите нужный COM-порт
		Baud:     9600,    // Вернули предыдущую скорость передачи
		Size:     8,
		Parity:   serial.ParityNone,
		StopBits: serial.Stop1,
	}

	// Открытие порта
	s, err := serial.OpenPort(config)
	if err != nil {
		log.Fatalf("Ошибка открытия порта: %v", err)
	}
	defer s.Close()

	fmt.Println("Порт открыт. Ожидаю команд.")

	buf := make([]byte, 1)

	for {

		n, err := s.Read(buf) // Читаем по 1 байту
		if err != nil {
			log.Printf("Ошибка чтения: %v", err)
			continue
		}

		if n > 0 {
			receivedByte := buf[0]

			if receivedByte&(1<<7) != 0 { // Если 7-й бит == 1, эмулируем 10-битное значение АЦП

				value := getADCValue()

				// Получаем два байта
				byteL, byteH := getLow6AndHigh4(value)

				fmt.Printf("RX: %b, BL:%b, BH:%b\n", receivedByte, byteL, byteH)
				time.Sleep(1 * time.Millisecond)

				s.Write([]byte{byteL})
				s.Write([]byte{byteH})

			} else {

				discretBtye := byte(rand.Intn(256))

				fmt.Printf("RX: %b, TX:%b\n", receivedByte, discretBtye)
				time.Sleep(1 * time.Millisecond)

				s.Write([]byte{discretBtye})
			}
		}
	}
}
