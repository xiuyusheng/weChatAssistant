package test

import (
	"io"
	"os"
	"testing"
	"time"

	"github.com/go-flac/flacpicture"
	"github.com/go-flac/flacvorbis"
	"github.com/go-flac/go-flac"
)

// flac添加图片
func TestAddFLACCover(t *testing.T) {
	img, _ := os.Open("../2430535878.jpg")
	imgData, _ := io.ReadAll(img)
	defer img.Close()
	fl, err := os.Open("../逆态度-张杰.flac")
	if err != nil {
		t.Log(err)
	}
	f, err := flac.ParseBytes(fl)
	if err != nil {
		panic(err)
	}

	picture, err := flacpicture.NewFromImageData(flacpicture.PictureTypeFrontCover, "Front cover", imgData, "image/jpeg")
	if err != nil {
		panic(err)
	}
	picturemeta := picture.Marshal()

	var cmts *flacvorbis.MetaDataBlockVorbisComment
	cmts = flacvorbis.New()
	cmts.Add(flacvorbis.FIELD_TITLE, "逆态度")
	cmts.Add(flacvorbis.FIELD_ARTIST, "张杰")
	cmts.Add(flacvorbis.FIELD_DATE, time.Now().Format("2006-01-02"))
	cmts.Add("LYRICS", "[00:0.00]逆态度 - 张杰 (Jason Zhang)")
	dd := cmts.Marshal()
	f.Meta = append(f.Meta, &picturemeta)
	f.Meta = append(f.Meta, &picturemeta, &dd)
	err = f.Save("../逆态度1.flac")
	if err != nil {
		t.Error(err)
	}
}
