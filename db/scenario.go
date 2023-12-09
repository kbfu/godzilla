package db

type Scenario struct {
	Base
	Name       string
	Definition string
}

func (*Scenario) TableName() string {
	return "scenario"
}

func (s *Scenario) GetByName() error {
	return Db.Where("name = ?", s.Name).Find(&s).Error
}
