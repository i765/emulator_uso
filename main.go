package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
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

	err := loadFile("analog.txt")
	if err != nil {
		fmt.Println("Error:", err)
	}

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

			fmt.Println("req:", receivedByte)

			if receivedByte&(1<<7) != 0 { // Проверяем 7-й бит (аналоговые или дискретные)
				// Обработка аналоговых каналов

				address := receivedByte & 0x7F // Обрезаем старший бит
				adcValue := getAnalogChannel(address)

				low6 := byte(0xC0 | (adcValue & 0x3F)) // 6 младших бит + 2 верхних бита = 11
				high4 := byte((adcValue >> 6) & 0x0F)  // 4 старших бита

				s.Write([]byte{low6, high4}) // Отправка ответа

			} else {
				// Обработка дискретных каналов
				// switcher := int(receivedByte>>6) & 0x01 // Бит A6 — коммутатор
				address := uint(receivedByte & 0x3F) // A5-A0 — адрес в группе

				s.Write([]byte{^discreteGroup_KD1[address]}) // Отправка ответа с инверсией
			}
		}

		time.Sleep(2 * time.Millisecond)

	}
}

func stripComment(line string) string {
	commentMarkers := []string{"#", "//", ";"}
	for _, marker := range commentMarkers {
		if idx := strings.Index(line, marker); idx != -1 {
			line = line[:idx]
		}
	}
	return strings.TrimSpace(line)
}

func loadFile(filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := stripComment(scanner.Text())
		if line == "" {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) < 2 {
			fmt.Printf("Skipping invalid line %d: %s\n", lineNum, line)
			continue
		}
		address, err1 := strconv.Atoi(parts[0])
		value, err2 := strconv.Atoi(parts[1])
		if err1 != nil || err2 != nil {
			fmt.Printf("Skipping line %d due to parse error: %s\n", lineNum, line)
			continue
		}
		setAnalogChannel(byte(address), uint16(value))
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading file: %w", err)
	}

	return nil
}
