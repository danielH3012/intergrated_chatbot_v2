install dulu library library yang dipakai dengan go get dan untuk react native pakai expo-speech dengan "npx expo install expo-speech"

untuk menggunakan key atau url bisa ganti di .eiai_go\.env dengan nama variabel BASE_AI_URL dan AI_KEY

frontend:
- react js: cd .ai-registration-frontend
            npm run dev

- react native: cd .react_native\eiai_react
                npm start

backend:
- eiai_go: cd .eiai_go
           go run main.go

- server\GO: cd .server\GO
             go run main.go
