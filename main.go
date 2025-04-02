package main

import (
	"fmt"
	"log"
	"time"

	"github.com/tarm/serial"
)

var discreteGroup_KD1 [48]byte // 48 дискретных групп крейт 1

var analogChannels [128]uint16 // Аналоговые каналы

// Функция установки аналогового канала
func setAnalogChannel(address byte, value uint16) {
	address--                               // Сдвигаем адрес на -1 (1-128)
	analogChannels[address] = value & 0x3FF // Ограничиваем 10 битами
}

// Функция получения аналогового канала
func getAnalogChannel(address byte) uint16 {

	return analogChannels[address]
}

func SetDiscreteBit(bitAddr uint, value bool) {

	// Вычисляем номер байта (0-47) и позицию бита (0-7)
	byteIndex := (bitAddr - 1) % 48 // Номер байта в массиве
	bitPos := (bitAddr - 1) / 48    // Позиция бита в байте

	if value {
		discreteGroup_KD1[byteIndex] |= 1 << bitPos // Установить бит в 1
	} else {
		discreteGroup_KD1[byteIndex] &^= 1 << bitPos // Установить бит в 0
	}

}

func main() {

	// Конфигурация аналоговых датчиков
	setAnalogChannel(1, 1000) // 6TL406P01
	setAnalogChannel(3, 1000) // 6TL602L02
	setAnalogChannel(4, 1000) // 6TW202F01
	setAnalogChannel(5, 1000) // 6TL102L02
	setAnalogChannel(6, 1000) // 6TW602F01

	// Конфигурация дискретных датчиков
	SetDiscreteBit(137, true)  // 6TC101L01
	SetDiscreteBit(106, true)  // 6TC102P01
	SetDiscreteBit(152, true)  // 6TC104L01
	SetDiscreteBit(150, false) // 6TC105P01
	SetDiscreteBit(153, true)  // 6TC106L01

	// Конфигурация COM-порта
	config := &serial.Config{
		Name:     "COM12", // Укажите нужный COM-порт
		Baud:     9600,    // Скорость передачи
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

	buf := make([]byte, 1) // Читаем по 1 байту
	for {
		n, err := s.Read(buf)
		if err != nil {
			log.Printf("Ошибка чтения: %v", err)
			continue
		}
		if n > 0 {
			receivedByte := buf[0]

			if receivedByte&(1<<7) != 0 { // Проверяем 7-й бит (аналоговые или дискретные)
				// Обработка аналоговых каналов

				address := receivedByte & 0x7F // Обрезаем старший бит
				adcValue := getAnalogChannel(address)

				low6 := byte(0xC0 | (adcValue & 0x3F)) // 6 младших бит + 2 верхних бита = 11
				high4 := byte((adcValue >> 6) & 0x0F)  // 4 старших бита

				s.Write([]byte{low6, high4}) // Отправка ответа

			} else {
				// Обработка дискретных каналов
				switcher := int(receivedByte>>6) & 0x01 // Бит A6 — коммутатор
				address := uint(receivedByte & 0x3F)    // A5-A0 — адрес в группе

				fmt.Println("switcher", switcher)
				fmt.Println("address", address)

				s.Write([]byte{^discreteGroup_KD1[address]}) // Отправка ответа с инверсией
			}
		}

		time.Sleep(2 * time.Millisecond)

	}
}
