// Application URLs
const DEFAULT_JAEGAR_URL = 'http://localhost:16686/'
const USER_SERVICE_URL = 'http://localhost:8080'
const PAYMENT_SERVICE_URL = 'http://localhost:8081'
const ORDER_SERVICE_URL = 'http://localhost:8082'

// Components
const Title = (props) => <p>{props.label}</p>
const Response = (props) => <pre>{JSON.stringify(props.result, null, 4)}</pre>
const ProductDD = (props) => {
  return (
    <select onChange={props.handleProductChange}>
      <option selected>Choose product</option>
      <option value={999}>MacBook Air</option>
      <option value={1100}>MacBook Pro</option>
      <option value={599}>Mac Mini</option>
    </select>
  )
}

const App = () => {
  let [user, setUser] = React.useState()
  let [payment, setPayment] = React.useState()
  let [order, setOrder] = React.useState()
  let [product, setProduct] = React.useState()
  let [jaegarurl, setJaegarUrl] = React.useState(DEFAULT_JAEGAR_URL)
  let createUser = async () => {
    try {
      const requestOptions = {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          username: 'Vaish',
          userid:getRandomInt(1,10000).toString(),
          account: 'savings',
        }),
      }
      let response = await fetch(`${USER_SERVICE_URL}/users`, requestOptions)
      let result = await response.json()
      setUser(result)
      console.log(result)
    } catch (error) {
      setUser(error.message)
      console.log('err', error.message)
    }
  }

  let handleProduct = (e) => {
    let { options, value } = e.target
    setProduct({
      product_name: options[options.selectedIndex].text,
      price: value,
    })
  }

  let getRandomInt = (min, max) => {
    min = Math.ceil(min);
    max = Math.floor(max);
    return Math.floor(Math.random() * (max - min + 1)) + min;
}

  let transferFund = async () => {
    console.log('transferFund', user)
    const requestOptions = {
      method: 'PUT',
	  headers: { 'Content-Type': 'application/json' },
	  body: JSON.stringify({
	    amount: 10000,
	  }),
    }

    let response = await fetch(
      `${PAYMENT_SERVICE_URL}/payments/transfer/id/${user.userid}`,
      requestOptions
    )
    let result = await response.json()
    setPayment(result)
    console.log(result)
  }

  let placeOrder = async () => {
    console.log('placeOrder', product)
    const requestOptions = {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        userid: user.userid,
        product_name: product.product_name,
        price: parseInt(product.price),
      }),
    }

    let response = await fetch(`${ORDER_SERVICE_URL}/orders`, requestOptions)
    let result = await response.json()
    setOrder(result)
    console.log(result)
  }

  let handleReset = () => {
    setUser()
    setPayment()
    setOrder()
    setProduct()
  }

  let setUrl = (e) => setJaegarUrl(e.target.value)
  let openJaegar = () => window.open(jaegarurl)
  return (
    <div>
      <button onClick={handleReset}>Reset Actions</button>
      <button onClick={openJaegar} style={{ marginLeft: 10, marginRight: 8 }}>
        Open Jaegar
      </button>
      <input
        placeholder="Type default jaegar url"
        type="text"
        onChange={setUrl}
        value={jaegarurl}
        style={{ width: 200 }}
      ></input>
      <p style={{ color: 'red' }}>Make sure jaegar is running on {jaegarurl}</p>
      <Title label="1. User Creation"></Title>
      <button onClick={createUser}>Create User</button>
      <Response result={user} />
      {user && (
        <div>
          <Title label="2. Transfer amount"></Title>
          <button onClick={transferFund}>Transfer Fund</button>
          <Response result={payment} />
        </div>
      )}
      {payment && (
        <div>
          <Title label="3. Place order"></Title>
          <ProductDD handleProductChange={handleProduct} />
          {product && <button onClick={placeOrder}>Place Order</button>}
          {order && (
            <div>
              <Response result={order} />
              <h3>Order Placed!</h3>
            </div>
          )}
        </div>
      )}
    </div>
  )
}

ReactDOM.render(<App />, document.getElementById('app'))
