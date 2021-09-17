package main

import (
	"encoding/csv"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/stripe/stripe-go"
	"github.com/stripe/stripe-go/issuing/card"
	"github.com/stripe/stripe-go/issuing/cardholder"
)

type Csv struct {
	Email        string
	Profilename  string
	Name         string
	Addressline1 string
	Addressline2 string
	City         string
	State        string
	Zipcode      string
	Country      string
	Phone        string
}

func ReadCsv(file string) ([][]string, error) {
	f, err := os.Open(file)
	if err != nil {
		return [][]string{}, err
	}
	defer f.Close()

	sheet := csv.NewReader(f)
	//skip line
	if _, err := sheet.Read(); err != nil {
		log.Fatal(err)
	}

	//read rest
	lines, err := sheet.ReadAll()
	if err != nil {
		return [][]string{}, err
	}
	return lines, nil
}

func CreateList(lines [][]string) []Csv {
	var result []Csv
	for _, line := range lines {
		data := Csv{
			Email:        line[0],
			Profilename:  line[1],
			Name:         line[3],
			Phone:        line[11],
			Addressline1: line[12],
			Addressline2: line[13],
			City:         line[16],
			State:        ConvertAc(line[17]), // create method
			Zipcode:      line[15],
		}
		result = append(result, data)
	}

	return result
}

func ConvertAc(state string) string {
	switch ac := state; ac {
	case "New Jersey":
		return "NJ"
	case "New York":
		return "NY"
	case "Pennsylvania":
		return "PA"
	case "California":
		return "CA"
	}
	return "null"
}

func CreateCardholder(name string, line1 string, line2 string, city string, state string, zipcode string, country string) *stripe.IssuingCardholder {
	stripe.Key = secretkey

	if line2 == "" { //test for empty string
		params := &stripe.IssuingCardholderParams{
			Billing: &stripe.IssuingCardholderBillingParams{
				Address: &stripe.AddressParams{
					Line1:      stripe.String(line1),
					City:       stripe.String(city),
					State:      stripe.String(state),
					Country:    stripe.String("US"),
					PostalCode: stripe.String(zipcode),
				},
			},
			Name:   stripe.String(name),
			Type:   stripe.String("individual"),
			Status: stripe.String("active"),
		}
		c, err := cardholder.New(params)
		if err != nil {
			log.Fatal(err)
		}
		return c
	} else {
		params := &stripe.IssuingCardholderParams{
			Billing: &stripe.IssuingCardholderBillingParams{
				Address: &stripe.AddressParams{
					Line1:      stripe.String(line1),
					Line2:      stripe.String(line2),
					City:       stripe.String(city),
					State:      stripe.String(state),
					Country:    stripe.String("US"),
					PostalCode: stripe.String(zipcode),
				},
			},
			Name:   stripe.String(name),
			Type:   stripe.String("individual"),
			Status: stripe.String("active"),
		}
		c, err := cardholder.New(params)
		if err != nil {
			log.Fatal(err)
		}
		return c
	}

}

func CreateLimit(x int64) *int64 {
	return &x
}

func CreateString(x string) *string {
	return &x
}
func CreateCard(profileinfo *stripe.IssuingCardholder) *stripe.IssuingCard {
	stripe.Key = secretkey

	spendinglimit := []*stripe.IssuingCardSpendingControlsSpendingLimitParams{}
	temp := &stripe.IssuingCardSpendingControlsSpendingLimitParams{
		Amount:   CreateLimit(1000000),
		Interval: CreateString("per_authorization"),
	}
	spendinglimit = append(spendinglimit, temp)

	params := &stripe.IssuingCardParams{
		Cardholder: stripe.String(profileinfo.ID),
		Currency:   stripe.String(string(stripe.CurrencyUSD)),
		SpendingControls: &stripe.IssuingCardSpendingControlsParams{
			SpendingLimits: spendinglimit,
		},
		Status: stripe.String("active"),
		Type:   stripe.String("virtual"),
	}

	c, err := card.New(params)
	if err != nil {
		log.Fatal(err)
	}
	return c
}

func GetCard(cardinfo *stripe.IssuingCard) (string, string) {
	stripe.Key = secretkey

	params := &stripe.IssuingCardParams{}
	params.AddExpand("number")
	params.AddExpand("cvc")
	c, err := card.Get(cardinfo.ID, params)
	if err != nil {
		log.Fatal(err)
	}

	return c.Number, c.CVC
}

var secretkey = ""

func main() {
	key, err := os.Open("../secretkey.txt")
	if err != nil {
		log.Fatal(err)
	}
	defer key.Close()
	temp, _ := ioutil.ReadAll(key)
	secretkey = string(temp)
	lines, err := ReadCsv("../Profiles.csv")
	if err != nil {
		log.Fatal(err)
	}

	list := CreateList(lines)

	result, err := os.Create("../Cards.csv")
	if err != nil {
		log.Fatal(err)
	}
	defer result.Close()

	writer := csv.NewWriter(result)
	writer.Write([]string{"Email Address", "Profile Name", "Only One Checkout", "Name on Card", "Card Type", "Card Number", "Expiration Month", "Expiration Year", "CVV", "Same Billing/Shipping", "Shipping Name", "Shipping Phone", "Shipping Address", "Shipping Address 2", "Shipping Address 3", "Shipping Post Code", "Shipping City", "Shipping State", "Shipping Country", "Billing Name", "Billing Phone", "Billing Address", "Billing Address 2", "Billing Address 3", "Billing Post Code", "Billing City", "Billing State", "Billing Country", "otherEntriesList", "Size (Optional)"})

	for i := range list {
		println("Genning Card for " + list[i].Profilename)
		println("\tName: "+list[i].Name+" Address: "+list[i].Addressline1+" "+list[i].Addressline2+" Zip: "+list[i].Zipcode, " State: "+list[i].State)
		user := CreateCardholder(list[i].Name, list[i].Addressline1, list[i].Addressline2, list[i].City, list[i].State, list[i].Zipcode, list[i].Country)
		card := CreateCard(user)
		cardnum, cvv := GetCard(card)
		writer.Write([]string{list[i].Email, list[i].Profilename, "FALSE", user.Name, "Visa", cardnum, strconv.FormatInt(card.ExpMonth, 10), strconv.FormatInt(card.ExpYear, 10), cvv, "TRUE", user.Name, list[i].Phone, user.Billing.Address.Line1, user.Billing.Address.Line2, "", user.Billing.Address.PostalCode, user.Billing.Address.City, user.Billing.Address.State, "United States", user.Name, list[i].Phone, user.Billing.Address.Line1, user.Billing.Address.Line2, "", user.Billing.Address.PostalCode, user.Billing.Address.City, user.Billing.Address.State, "United States", "[]"})
		time.Sleep(3 * time.Second)
	}

	writer.Flush()
	err = writer.Error()
	if err != nil {
		log.Fatal(err)
	}

	println("Card Gen Finished :)")
}
