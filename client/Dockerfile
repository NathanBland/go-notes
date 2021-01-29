FROM node:lts-alpine 

WORKDIR /go-notes-client
ADD ./package.json ./package.json
ADD ./package-lock.json ./package-lock.json

RUN npm ci

CMD [ "vite build" ]
